package sqlite_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/store/sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	// We need to use a shared cache. Otherwise, if all connections close, the
	// underlying database disappears.
	db := sqlite.New("file::memory:?cache=shared")

	if err := db.InsertEntry(bytes.NewBufferString("hello, world!"), types.UploadMetadata{
		ID:       types.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entry, err := db.GetEntry(types.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to get entry from DB: %v", err)
	}

	contents, err := ioutil.ReadAll(entry.Reader)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	expected := "hello, world!"
	if string(contents) != expected {
		log.Fatalf("unexpected file contents: got %v, want %v", string(contents), expected)
	}

	meta, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if len(meta) != 1 {
		t.Fatalf("unexpected metadata size: got %v, want %v", len(meta), 1)
	}

	if meta[0].Size != len(expected) {
		t.Fatalf("unexpected file size in entry metadata: got %v, want %v", meta[0].Size, len(expected))
	}

	expectedFilename := types.Filename("dummy-file.txt")
	if meta[0].Filename != expectedFilename {
		t.Fatalf("unexpected filename: got %v, want %v", meta[0].Filename, expectedFilename)
	}

	err = db.DeleteEntry(types.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to delete entry: %v", err)
	}

	meta, err = db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if len(meta) != 0 {
		t.Fatalf("unexpected metadata size: got %v, want %v", len(meta), 0)
	}
}

func mustParseExpirationTime(s string) types.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}
