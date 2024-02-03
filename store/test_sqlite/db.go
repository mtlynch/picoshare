package test_sqlite

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

const optimizeForLitestream = false

func New() sqlite.Store {
	return sqlite.New(ephemeralDbURI(), optimizeForLitestream)
}

func NewWithChunkSize(chunkSize int) sqlite.Store {
	return sqlite.NewWithChunkSize(ephemeralDbURI(), chunkSize, optimizeForLitestream)
}

func ephemeralDbURI() string {
	name := random.String(
		10,
		[]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}
