package file_test

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/sqlite/file"
)

type (
	mockChunkRow struct {
		id         picoshare.EntryID
		chunkIndex int
		chunk      []byte
	}

	mockSqlDB struct {
		rows []mockChunkRow
		err  error
	}
)

var errMockSqlFailure = errors.New("fake SQL error")

func (db *mockSqlDB) Exec(query string, args ...any) (sql.Result, error) {
	id, err := picoshare.NewEntryID(args[0].(sql.NamedArg).Value.(string))
	if err != nil {
		return nil, err
	}
	chunkIndex := args[1].(sql.NamedArg).Value.(int)
	chunk := args[2].(sql.NamedArg).Value.([]byte)
	chunkCopy := make([]byte, len(chunk))
	copy(chunkCopy, chunk)
	db.rows = append(db.rows, mockChunkRow{
		id:         id,
		chunkIndex: chunkIndex,
		chunk:      chunkCopy,
	})
	return nil, db.err
}

func TestWriteFile(t *testing.T) {
	entryID := picoshare.MustCreateEntryID("abcdefghij")
	for _, tt := range []struct {
		explanation  string
		id           picoshare.EntryID
		data         []byte
		chunkSize    uint64
		sqlExecErr   error
		errExpected  error
		rowsExpected []mockChunkRow
	}{
		{
			explanation: "data is smaller than chunk size",
			id:          entryID,
			data:        []byte("hello, world!"),
			chunkSize:   25,
			rowsExpected: []mockChunkRow{
				{
					id:         entryID,
					chunkIndex: 0,
					chunk:      []byte("hello, world!"),
				},
			},
		},
		{
			explanation: "data fits exactly in single chunk",
			id:          entryID,
			data:        []byte("01234"),
			chunkSize:   5,
			rowsExpected: []mockChunkRow{
				{
					id:         entryID,
					chunkIndex: 0,
					chunk:      []byte("01234"),
				},
			},
		},
		{
			explanation: "data occupies a partial chunk after the first",
			id:          entryID,
			data:        []byte("0123456"),
			chunkSize:   5,
			rowsExpected: []mockChunkRow{
				{
					id:         entryID,
					chunkIndex: 0,
					chunk:      []byte("01234"),
				},
				{
					id:         entryID,
					chunkIndex: 1,
					chunk:      []byte("56"),
				},
			},
		},
		{
			explanation: "data spans exactly two chunks",
			id:          entryID,
			data:        []byte("0123456789"),
			chunkSize:   5,
			rowsExpected: []mockChunkRow{
				{
					id:         entryID,
					chunkIndex: 0,
					chunk:      []byte("01234"),
				},
				{
					id:         entryID,
					chunkIndex: 1,
					chunk:      []byte("56789"),
				},
			},
		},
		{
			explanation: "write fails when SQL transaction returns error",
			id:          entryID,
			data:        []byte("0123456789"),
			chunkSize:   5,
			sqlExecErr:  errMockSqlFailure,
			errExpected: errMockSqlFailure,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			tx := mockSqlDB{
				err: tt.sqlExecErr,
			}

			w := file.NewWriter(&tx, tt.id, tt.chunkSize)
			n, err := w.Write(tt.data)

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", err, tt.errExpected)
			}
			if err != nil {
				return
			}
			if got, want := n, len(tt.data); got != want {
				t.Errorf("n=%d, want=%d", got, want)
			}

			if err := w.Close(); err != nil {
				t.Errorf("failed to close writer: %v", err)
			}

			if got, want := tx.rows, tt.rowsExpected; !reflect.DeepEqual(got, want) {
				t.Errorf("rows=%v, want %v", tx.rows, tt.rowsExpected)
			}
		})
	}
}
