package sqlite_test

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

func New() store.Store {
	name := random.String(10, []rune("abcdefghijklmnopqrstuvwxyz0123456789"))

	return sqlite.New(fmt.Sprintf("file:%s?mode=memory&cache=shared", name))
}
