package file_test

import (
	"database/sql"
	"testing"

	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
	"github.com/mtlynch/picoshare/v2/types"
)

type mockSqlDB struct {
}

func (db mockSqlDB) Prepare(string) (*sql.Stmt, error) {
	return &sql.Stmt{}, nil
}

func TestUploadValidFile(t *testing.T) {
	db := mockSqlDB{}

	cr, err := file.NewReader(&db, types.EntryID("1"))
	if err != nil {
		t.Fatalf("failed to create chunk reader: %v", err)
	}

	var buf []byte
	n, err := cr.Read(buf)
	if err != nil {
		t.Fatalf("failed to read DB: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("unexpected read size: got %d, want %d", n, len(buf))
	}
}
