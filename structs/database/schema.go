package database

import (
	"database/sql"
	"errors"
	"strings"
)

// Column describes one column. Name is quoted for you; Type and Constraints
// are raw SQL and must come from your code, never from user input.
//
//	database.Column{Name: "id", Type: "INTEGER", Constraints: "PRIMARY KEY"}
//	database.Column{Name: "email", Type: "TEXT", Constraints: "NOT NULL DEFAULT ''"}
type Column struct {
	Name        string
	Type        string
	Constraints string
}

// quoteIdent quotes a table or column name so it can't break out of the SQL.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func (c Column) def() string {
	parts := []string{quoteIdent(c.Name)}
	if c.Type != "" {
		parts = append(parts, c.Type)
	}
	if c.Constraints != "" {
		parts = append(parts, c.Constraints)
	}
	return strings.Join(parts, " ")
}

// CreateTable creates the table if it doesn't exist yet.
func (d *DB) CreateTable(table string, cols ...Column) error {
	if table == "" || len(cols) == 0 {
		return errors.New("database: CreateTable needs a table name and at least one column")
	}
	defs := make([]string, len(cols))
	for i, c := range cols {
		if c.Name == "" {
			return errors.New("database: column with empty name")
		}
		defs[i] = c.def()
	}
	_, err := d.Exec("CREATE TABLE IF NOT EXISTS " + quoteIdent(table) +
		" (" + strings.Join(defs, ", ") + ")")
	return err
}

// AddColumn adds a column to an existing table. It does nothing if a column
// with that name (case-insensitive) is already there, so it is safe to call
// on every startup and from several goroutines.
//
// SQLite limits what ALTER TABLE ADD COLUMN accepts: no PRIMARY KEY or UNIQUE,
// and NOT NULL needs a DEFAULT.
func (d *DB) AddColumn(table string, col Column) error {
	if table == "" || col.Name == "" {
		return errors.New("database: AddColumn needs a table and column name")
	}
	// The check and the ALTER run in one write transaction, so two callers
	// can't both decide the column is missing.
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var one int
	err = tx.QueryRow(
		"SELECT 1 FROM pragma_table_info(?) WHERE name = ? COLLATE NOCASE",
		table, col.Name,
	).Scan(&one)
	switch {
	case err == nil:
		return nil // already exists
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}

	if _, err := tx.Exec("ALTER TABLE " + quoteIdent(table) + " ADD COLUMN " + col.def()); err != nil {
		return err
	}
	return tx.Commit()
}

// HasColumn reports whether the table has a column with that name.
func (d *DB) HasColumn(table, column string) (bool, error) {
	rows, err := d.Query(
		"SELECT 1 FROM pragma_table_info(?) WHERE name = ? COLLATE NOCASE",
		table, column,
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	found := rows.Next()
	return found, rows.Err()
}












































