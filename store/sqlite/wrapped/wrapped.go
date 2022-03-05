package wrapped

import "database/sql"

type SqlTx interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

func New(tx *sql.Tx) SqlTx {
	return tx
}
