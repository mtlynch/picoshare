package file_test

import (
	_ "github.com/mattn/go-sqlite3"
)

/*
func TestReadFile(t *testing.T) {
	db := sqlite.New(":memory:")
	id := types.EntryID("dummyid")
	fw, err := file.NewWriter(&db, id, 10)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}

	fr, err := file.NewReader(&db, types.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to create chunk reader: %v", err)
	}

	var buf []byte
	n, err := fr.Read(buf)
	if err != nil {
		t.Fatalf("failed to read DB: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("unexpected read size: got %d, want %d", n, len(buf))
	}
}
*/
