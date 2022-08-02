package wrapped

import "database/sql"

// Interface to represent a sql.DB object that can be mocked for testing.
type SqlDB interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

// Create a new SqlDB instance by wrapping a sql.DB instance.
func New(db *sql.DB) SqlDB {
	return db
}
