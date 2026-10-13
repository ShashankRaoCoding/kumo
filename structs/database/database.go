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
func New() *Database {
	conn, err := sql.Open("sqlite", "file:kumo.db?_pragma=foreign_keys(1)")
	if err != nil {
		panic(err)
	}

	return &Database{db: conn}
}

// Close closes the underlying DB connection.
func (db *Database) Close() error {
	if db == nil || db.db == nil {
		return nil
	}
	return db.db.Close()
}
