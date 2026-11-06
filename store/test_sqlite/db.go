package test_sqlite

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/random"
	"github.com/mtlynch/picoshare/store/sqlite"
)

const optimizeForLitestream = false

func New(t testing.TB) sqlite.Store {
	t.Helper()
	return newWithChunkSize(0)
}

func NewWithChunkSize(t testing.TB, chunkSize uint64) sqlite.Store {
	t.Helper()
	return newWithChunkSize(chunkSize)
}

func newWithChunkSize(chunkSize uint64) sqlite.Store {
	return sqlite.New(sqlite.Params{
		Path:                  ephemeralDbURI(),
		OptimizeForLitestream: optimizeForLitestream,
		Now: func() time.Time {
			return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		},
		PicoShareChunkSize: chunkSize,
	})
}

func ephemeralDbURI() string {
	name := random.String(
		10,
		[]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}
