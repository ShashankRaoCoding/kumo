package database

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

// Database wraps a sql.DB connection.
type Database struct {
	db *sql.DB
}

// New creates a sqlite database connection using the modernc driver.
// It accepts an optional filepath argument. If no argument is provided,
// it defaults to "file:kumo.db?_pragma=foreign_keys(1)?_pragma=journal_mode(WAL)".
func New(path ...string) (*Database, error) {
	dsn := "file:kumo.db?_pragma=foreign_keys(1)?_pragma=journal_mode(WAL)"
	if len(path) > 0 && path[0] != "" {
		candidate := path[0]
		switch {
		case strings.HasPrefix(candidate, "file:") || strings.Contains(candidate, "?"):
			dsn = candidate
		default:
			dsn = "file:" + candidate + "?_pragma=foreign_keys(1)?_pragma=journal_mode(WAL)"
		}
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	return &Database{db: conn}, nil
}

func (db *Database) InitialiseSchema(schema string) error {
	if db == nil || db.db == nil {
		return nil
	}

	_, err := db.db.Exec(schema)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the underlying DB connection.
func (db *Database) Close() error {
	if db == nil || db.db == nil {
		return nil
	}
	return db.db.Close()
}
