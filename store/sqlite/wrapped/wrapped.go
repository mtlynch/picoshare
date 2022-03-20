package wrapped

import "database/sql"

// Interface to represent a sql.Tx object that can be mocked for testing.
type SqlTx interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

// Create a new SqlTx instance by wrapping a sql.Tx instance.
func New(tx *sql.Tx) SqlTx {
	return tx
}
