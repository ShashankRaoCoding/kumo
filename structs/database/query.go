package database

// Query executes a SQL query against the database.
func (db *Database) Query(query string) Rows {
	if db == nil || db.db == nil {
		return Rows{err: errDatabaseNotInitialized}
	}

	rows, err := db.db.Query(query)
	return Rows{rows: rows, err: err}
}
