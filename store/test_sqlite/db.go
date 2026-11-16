package test_sqlite

import (
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/vfs/memdb"

	"github.com/mtlynch/picoshare/store/sqlite"
)

const optimizeForLitestream = false

func New(t testing.TB) sqlite.Store {
	t.Helper()
	return newWithChunkSize(t, 0)
}

func NewWithChunkSize(t testing.TB, chunkSize uint64) sqlite.Store {
	t.Helper()
	return newWithChunkSize(t, chunkSize)
}

func newWithChunkSize(t testing.TB, chunkSize uint64) sqlite.Store {
	t.Helper()

	return sqlite.New(sqlite.Params{
		Path:                  memdb.TestDB(t),
		OptimizeForLitestream: optimizeForLitestream,
		Now: func() time.Time {
			return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		},
		PicoShareChunkSize: chunkSize,
	})
}
