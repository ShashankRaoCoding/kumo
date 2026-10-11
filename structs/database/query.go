package database

import (
	"context"
	"database/sql"
)

// Rows is the result of a Query. It wraps *sql.Rows, so the usual
// Next/Scan/Close loop works, and adds helpers for reading whole rows
// without knowing the columns in advance.
//
//	rows, err := db.Query(`select * from "users"`)
//	if err != nil { ... }
//	defer rows.Close()
//	for rows.Next() {
//		row, _ := rows.Map()
//		fmt.Println(row["name"])
//	}
type Rows struct {
	rows *sql.Rows
}

// Query runs a query that returns rows. The caller must close the Rows.
// Close all Rows before calling DB.Close.
func (d *DB) Query(query string, args ...any) (*Rows, error) {
	return d.QueryContext(context.Background(), query, args...)
}

// QueryContext is Query with a context.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	if err := d.begin(); err != nil {
		return nil, err
	}
	defer d.mu.RUnlock()

	r, err := d.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows: r}, nil
}

// Next advances to the next row. It returns false when there are no more
// rows or an error occurred; check Err afterwards.
func (r *Rows) Next() bool { return r.rows.Next() }

// Scan copies the columns of the current row into dest.
func (r *Rows) Scan(dest ...any) error { return r.rows.Scan(dest...) }

// Columns returns the column names of the result.
func (r *Rows) Columns() ([]string, error) { return r.rows.Columns() }

// Err returns any error hit while iterating.
func (r *Rows) Err() error { return r.rows.Err() }

// Close releases the rows. It is safe to call more than once.
func (r *Rows) Close() error { return r.rows.Close() }

// Map reads the current row into a Row keyed by column name.
// Call it after Next returns true.
func (r *Rows) Map() (Row, error) {
	cols, err := r.rows.Columns()
	if err != nil {
		return nil, err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := r.rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	row := make(Row, len(cols))
	for i, c := range cols {
		row[c] = vals[i]
	}
	return row, nil
}

// All reads every remaining row into a slice and closes the Rows.
func (r *Rows) All() ([]Row, error) {
	defer r.rows.Close()
	var out []Row
	for r.rows.Next() {
		row, err := r.Map()
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, r.rows.Err()
}












































