package file_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
	"github.com/mtlynch/picoshare/v2/types"
)

type (
	mockChunkRow struct {
		id         types.EntryID
		chunkIndex int
		chunk      []byte
	}

	mockSqlTx struct {
		rows []mockChunkRow
	}
)

func (tx *mockSqlTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	tx.rows = append(tx.rows, mockChunkRow{
		id:         args[0].(types.EntryID),
		chunkIndex: args[1].(int),
		chunk:      args[2].([]byte),
	})
	return nil, nil
}

func TestWriteFileThatFitsInSingleChunk(t *testing.T) {
	tx := mockSqlTx{}
	chunkSize := 20

	w := file.NewWriter(&tx, types.EntryID("1"), chunkSize)
	data := []byte("hello, world!")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	if n != len(data) {
		t.Fatalf("wrong size data written: got %d, want %d", n, len(data))
	}

	rowsExpected := []mockChunkRow{
		{
			id:         types.EntryID("1"),
			chunkIndex: 0,
			chunk:      []byte("hello, world!"),
		},
	}
	if !reflect.DeepEqual(tx.rows, rowsExpected) {
		t.Fatalf("unexpected DB transactions: got %v, want %v", tx.rows, rowsExpected)
	}
}

func TestWriteFileThatSpansTwoChunks(t *testing.T) {
	tx := mockSqlTx{}
	chunkSize := 5

	w := file.NewWriter(&tx, types.EntryID("1"), chunkSize)
	data := []byte("0123456789")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	if n != len(data) {
		t.Fatalf("wrong size data written: got %d, want %d", n, len(data))
	}

	rowsExpected := []mockChunkRow{
		{
			id:         types.EntryID("1"),
			chunkIndex: 0,
			chunk:      []byte("01234"),
		},
		{
			id:         types.EntryID("1"),
			chunkIndex: 1,
			chunk:      []byte("56789"),
		},
	}
	if !reflect.DeepEqual(tx.rows, rowsExpected) {
		t.Fatalf("unexpected DB transactions: got %v, want %v", tx.rows, rowsExpected)
	}
}

// TODO: Handle multiple writes that fill up part of a buffer
