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

	mu     sync.RWMutex // guards closed; held (read) for the duration of each call
	closed bool

	reader *sql.DB // many connections
	writer *sql.DB // exactly one connection (same *sql.DB as reader for :memory:)
	err    error   // open error, reported on every use
}

// buildDSN adds the pragmas every connection in the pool needs.
func buildDSN(path string, readOnly bool) string {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	if !isMemory(path) {
		q.Add("_pragma", "journal_mode(WAL)")
	}
	if readOnly {
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
func New(path string) *DB {
	d := &DB{Path: path}

	if isMemory(path) {
		// Every connection to a plain :memory: DB is a *separate* database,
		// so share one connection for both reads and writes.
		conn, err := open(buildDSN(":memory:", false), 1)
		if err != nil {
			d.err = fmt.Errorf("open %s: %w", path, err)
			return d
		}
		d.reader, d.writer = conn, conn
		return d
	}

	w, err := open(buildDSN(path, false), 1)
	if err != nil {
		d.err = fmt.Errorf("open %s: %w", path, err)
		return d
	}
	r, err := open(buildDSN(path, true), 8)
	if err != nil {
		w.Close()
		d.err = fmt.Errorf("open %s (readers): %w", path, err)
		return d
	}
	d.writer, d.reader = w, r
	return d
}

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

// Query runs a query that returns rows. The caller must close the rows.
// Close all rows before calling DB.Close.
func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return d.QueryContext(context.Background(), query, args...)
}

// QueryContext is Query with a context.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if err := d.begin(); err != nil {
		return nil, err
	}
	defer d.mu.RUnlock()
	return d.reader.QueryContext(ctx, query, args...)
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












































