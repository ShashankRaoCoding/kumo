// Package database is a concurrency-safe wrapper around modernc.org/sqlite
// (pure Go, no CGO).
//
// Each call to New returns an independent handle, so you can open as many
// databases as you like:
//
//	db := database.New("database1.db")
//	rows, err := db.Query("select * from users")
//	db2 := database.New("database2.db")
//
// A *DB is safe for use by many goroutines at once. Under the hood:
//
//   - WAL journal mode lets readers run while a write is in progress.
//   - Reads (Query, QueryRow) use a pool of connections.
//   - Writes (Exec, Begin) go through a single dedicated connection, so
//     writers queue up in Go instead of failing with "database is locked".
//   - busy_timeout makes SQLite wait for locks held by other processes.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// ErrClosed is returned when a DB is used after Close.
var ErrClosed = errors.New("database: closed")

// DB wraps one SQLite file with a read pool and a single writer connection.
type DB struct {
	Path string

	created bool // true if New created the file just now

	mu     sync.RWMutex // guards closed; held (read) for the duration of each call
	closed bool

	reader *sql.DB // many connections
	writer *sql.DB // exactly one connection (same *sql.DB as reader for :memory:)
	err    error   // open error, reported on every use
}

// config holds the settings New builds from its options.
type config struct {
	pragmas    []string // extra pragmas, applied to every connection
	maxReaders int
	init       func(*DB) error // runs only when the file was just created
}

// WithInit registers setup that runs only when the database file did not
// exist (or was empty) and has just been created, e.g. creating tables.
// On later starts the file is simply opened and fn is skipped.
//
// If fn returns an error, the half-initialised file is deleted so the next
// New will run fn again, and the error is reported by Err and every method.
func WithInit(fn func(*DB) error) Option {
	return func(c *config) { c.init = fn }
}

// Option customizes New.
type Option func(*config)

// WithPragma adds pragmas applied to every connection (readers and writer),
// written as in SQL without the "pragma" keyword, e.g. "cache_size(-64000)".
// They are applied after the defaults, so they can override them, e.g.
// WithPragma("synchronous(FULL)").
func WithPragma(pragmas ...string) Option {
	return func(c *config) { c.pragmas = append(c.pragmas, pragmas...) }
}

// WithMaxReaders sets the size of the read pool (default 8).
func WithMaxReaders(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxReaders = n
		}
	}
}

// buildDSN adds the pragmas every connection in the pool needs.
func buildDSN(path string, readOnly bool, extra []string) string {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	if !isMemory(path) {
		q.Add("_pragma", "journal_mode(WAL)")
	}
	for _, p := range extra {
		q.Add("_pragma", strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "pragma ")))
	}
	if readOnly {
		// Last, so an option can't turn the reader pool writable.
		q.Add("_pragma", "query_only(1)")
	}
	return (&url.URL{Opaque: "file:" + escapePath(path), RawQuery: q.Encode()}).String()
}

// escapePath protects the characters that would be read as URI syntax.
func escapePath(p string) string {
	return strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(p)
}

func isMemory(path string) bool {
	return path == ":memory:" || path == "" || strings.Contains(path, "mode=memory")
}

// New opens (or creates) the SQLite database at path.
// If opening fails, the error is returned by every later method call
// instead of panicking, so New keeps its single-return signature.
func New(path string, opts ...Option) *DB {
	d := &DB{Path: path}

	cfg := config{maxReaders: 8}
	for _, opt := range opts {
		opt(&cfg)
	}

	if isMemory(path) {
		// Every connection to a plain :memory: DB is a *separate* database,
		// so share one connection for both reads and writes.
		memPath := path
		if memPath == "" {
			memPath = ":memory:"
		}
		conn, err := open(buildDSN(memPath, false, cfg.pragmas), 1)
		if err != nil {
			d.err = fmt.Errorf("open %s: %w", path, err)
			return d
		}
		d.reader, d.writer = conn, conn
		return d
	}

	// Create the parent folder if needed, and note whether the file is new.
	if dir := filepath.Dir(path); dir != "." && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			d.err = fmt.Errorf("create dir for %s: %w", path, err)
			return d
		}
	}
	if !strings.HasPrefix(path, "file:") {
		info, statErr := os.Stat(path)
		d.created = statErr != nil || info.Size() == 0
	}

	w, err := open(buildDSN(path, false, cfg.pragmas), 1)
	if err != nil {
		d.err = fmt.Errorf("open %s: %w", path, err)
		return d
	}
	r, err := open(buildDSN(path, true, cfg.pragmas), cfg.maxReaders)
	if err != nil {
		w.Close()
		d.err = fmt.Errorf("open %s (readers): %w", path, err)
		return d
	}
	d.writer, d.reader = w, r

	if d.created && cfg.init != nil {
		if err := cfg.init(d); err != nil {
			d.reader.Close()
			d.writer.Close()
			// Remove the half-built file so the next start runs init again.
			for _, suffix := range []string{"", "-wal", "-shm"} {
				os.Remove(path + suffix)
			}
			d.err = fmt.Errorf("init %s: %w", path, err)
			return d
		}
	}
	return d
}

// Created reports whether New created the database file (true on first run,
// false when an existing database was opened).
func (d *DB) Created() bool { return d.created }

func open(dsn string, maxConns int) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(maxConns)
	conn.SetMaxIdleConns(maxConns)
	conn.SetConnMaxLifetime(0)
	// Force the file to actually be opened so problems surface early.
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// begin checks state and takes the read lock; callers must call d.mu.RUnlock.
func (d *DB) begin() error {
	d.mu.RLock()
	if d.err != nil {
		d.mu.RUnlock()
		return d.err
	}
	if d.closed {
		d.mu.RUnlock()
		return ErrClosed
	}
	return nil
}

// Exec runs a statement that doesn't return rows (INSERT, CREATE, ...).
// All writes are serialized through one connection.
func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	return d.ExecContext(context.Background(), query, args...)
}

// ExecContext is Exec with a context.
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := d.begin(); err != nil {
		return nil, err
	}
	defer d.mu.RUnlock()
	return d.writer.ExecContext(ctx, query, args...)
}

// Begin starts a write transaction. It holds the single writer connection
// until Commit or Rollback, so keep transactions short and always finish them.
func (d *DB) Begin() (*sql.Tx, error) {
	return d.BeginTx(context.Background(), nil)
}

// BeginTx is Begin with a context and options.
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if err := d.begin(); err != nil {
		return nil, err
	}
	defer d.mu.RUnlock()
	return d.writer.BeginTx(ctx, opts)
}

// Err returns any error that occurred while opening the database.
func (d *DB) Err() error { return d.err }

// Close releases all connections. It is safe to call more than once and
// from multiple goroutines; later calls on the DB return ErrClosed.
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	if d.closed {
		return nil
	}
	d.closed = true

	// Checkpoint the WAL so the .db file is self-contained after close.
	if !isMemory(d.Path) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		d.writer.ExecContext(ctx, "pragma wal_checkpoint(TRUNCATE)")
		cancel()
	}

	var errs []error
	if d.reader != d.writer {
		errs = append(errs, d.reader.Close())
	}
	errs = append(errs, d.writer.Close())
	return errors.Join(errs...)
}












































