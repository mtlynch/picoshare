package sqlite_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	chunkSize := 5
	db := test_sqlite.NewWithChunkSize(chunkSize)

	contents := "hello, world!"

	if err := db.InsertEntry(bytes.NewBufferString(contents), types.UploadMetadata{
		ID:       types.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Size:     int64(len(contents)),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entry, err := db.GetEntry(types.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to get entry from DB: %v", err)
	}

	data, err := ioutil.ReadAll(entry.Reader)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	if got, want := string(data), contents; got != want {
		t.Fatalf("contents=%s, want=%s", got, want)
	}

	meta, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if len(meta) != 1 {
		t.Fatalf("unexpected metadata size: got %v, want %v", len(meta), 1)
	}

	if got, want := meta[0].Size, int64(len(contents)); got != want {
		t.Fatalf("file size=%d, want %d", got, want)
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

func TestReadLastByteOfEntry(t *testing.T) {
	chunkSize := 5
	db := test_sqlite.NewWithChunkSize(chunkSize)

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

	pos, err := entry.Reader.Seek(1, io.SeekEnd)
	if err != nil {
		t.Fatalf("failed to seek file reader: %v", err)
	}

	expectedPos := int64(12)
	if pos != expectedPos {
		t.Fatalf("unexpected file position: got %d, want %d", pos, expectedPos)
	}

	contents, err := ioutil.ReadAll(entry.Reader)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	expected := "!"
	if string(contents) != expected {
		log.Fatalf("unexpected file contents: got %v, want %v", string(contents), expected)
	}
}

func mustParseExpirationTime(s string) types.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}
