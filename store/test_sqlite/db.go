package test_sqlite

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

func New() store.Store {
	return sqlite.New(ephemeralDbURI())
}

func NewWithChunkSize(chunkSize int) store.Store {
	return sqlite.NewWithChunkSize(ephemeralDbURI(), chunkSize)
}

func ephemeralDbURI() string {
	name := random.String(
		10,
		[]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}
