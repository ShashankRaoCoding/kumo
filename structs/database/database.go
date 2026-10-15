package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Database wraps a sql.DB connection.
type Database struct {
	db *sql.DB
}

// New creates a sqlite database connection using the modernc driver.
// It accepts an optional filepath argument. If no argument is provided,
// it defaults to "file:kumo.db?_pragma=foreign_keys(1)".
func New(path ...string) (*Database, error) {
	// default DSN
	dsn := "file:kumo.db?_pragma=foreign_keys(1)"
	if len(path) > 0 && path[0] != "" {
		dsn = path[0]
	}

	conn, err := sql.Open("sqlite", dsn)
	return &Database{db: conn}, err
}

// Close closes the underlying DB connection.
func (db *Database) Close() error {
	if db == nil || db.db == nil {
		return nil
	}
	return db.db.Close()
}
