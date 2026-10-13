package database

import "database/sql"

var errDatabaseNotInitialized = &databaseError{"database is not initialized"}

// Rows wraps sql.Rows and exposes simple accessors.
type Rows struct {
	rows *sql.Rows
	err  error
}

// Close closes the result set.
func (r Rows) Close() error {
	if r.rows == nil {
		return nil
	}
	return r.rows.Close()
}

// Next advances to the next row.
func (r Rows) Next() bool {
	if r.rows == nil {
		return false
	}
	return r.rows.Next()
}

// Scan copies the current row values into dest.
func (r Rows) Scan(dest ...any) error {
	if r.rows == nil {
		return r.err
	}
	return r.rows.Scan(dest...)
}

// Err returns the last error encountered.
func (r Rows) Err() error {
	if r.err != nil {
		return r.err
	}
	if r.rows == nil {
		return nil
	}
	return r.rows.Err()
}

type databaseError struct {
	message string
}

func (e *databaseError) Error() string {
	return e.message
}
