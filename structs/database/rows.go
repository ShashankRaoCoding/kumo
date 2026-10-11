package database

import (
	"errors"
	"sort"
	"strings"
)

// Row maps column names to values for Insert.
type Row map[string]any

// buildInsert returns the INSERT statement and its arguments. Column names
// are quoted and sorted (so the SQL is stable); values are always passed as
// parameters, never spliced into the SQL.
func buildInsert(table string, row Row) (string, []any) {
	if len(row) == 0 {
		return "INSERT INTO " + quoteIdent(table) + " DEFAULT VALUES", nil
	}
	names := make([]string, 0, len(row))
	for name := range row {
		names = append(names, name)
	}
	sort.Strings(names)

	cols := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		cols[i] = quoteIdent(name)
		args[i] = row[name]
	}
	query := "INSERT INTO " + quoteIdent(table) +
		" (" + strings.Join(cols, ", ") + ") VALUES (" +
		strings.TrimSuffix(strings.Repeat("?, ", len(names)), ", ") + ")"
	return query, args
}

// Insert adds one row and returns its rowid (the INTEGER PRIMARY KEY value
// if the table has one).
//
//	id, err := db.Insert("users", database.Row{"name": "Ada", "email": "ada@example.com"})
func (d *DB) Insert(table string, row Row) (int64, error) {
	if table == "" {
		return 0, errors.New("database: Insert needs a table name")
	}
	query, args := buildInsert(table, row)
	res, err := d.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertMany adds several rows in one transaction: either all of them are
// stored or none are. Rows may use different columns.
func (d *DB) InsertMany(table string, rows []Row) error {
	if table == "" {
		return errors.New("database: InsertMany needs a table name")
	}
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, row := range rows {
		query, args := buildInsert(table, row)
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}












































