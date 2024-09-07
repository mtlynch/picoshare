package sqlite_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	db := test_sqlite.New()

	inputContents := "hello, world!"

	if err := db.InsertEntry(bytes.NewBufferString(inputContents), picoshare.UploadMetadata{
		ID:       picoshare.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Size:     uint64(len(inputContents)),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entry, err := db.GetEntryMetadata(picoshare.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to get entry from DB: %v", err)
	}

	var entryContents bytes.Buffer
	db.ReadEntryFile(entry.ID, func(reader io.ReadSeeker) {
		if _, err := io.Copy(&entryContents, reader); err != nil {
			t.Fatalf("failed to read entry contents: %v", err)
		}
	})

	expectedContents := "hello, world!"
	if got, want := entryContents.String(), expectedContents; got != want {
		t.Errorf("stored contents=%v, want=%v", got, want)
	}

	meta, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if len(meta) != 1 {
		t.Fatalf("unexpected metadata size: got %v, want %v", len(meta), 1)
	}

	if meta[0].Size != uint64(len(expectedContents)) {
		t.Fatalf("unexpected file size in entry metadata: got %v, want %v", meta[0].Size, len(expectedContents))
	}

	if meta[0].DownloadCount != 0 {
		t.Fatalf("unexpected download count in entry metadata: got %v, want %v", meta[0].DownloadCount, 0)
	}

	expectedFilename := picoshare.Filename("dummy-file.txt")
	if meta[0].Filename != expectedFilename {
		t.Fatalf("unexpected filename: got %v, want %v", meta[0].Filename, expectedFilename)
	}

	err = db.DeleteEntry(picoshare.EntryID("dummy-id"))
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

/*
func TestReadLastByteOfEntry(t *testing.T) {
	db := test_sqlite.New()

	if err := db.InsertEntry(bytes.NewBufferString("hello, world!"), picoshare.UploadMetadata{
		ID:       picoshare.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entry, err := db.GetEntryMetadata(picoshare.EntryID("dummy-id"))
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

	contents, err := io.ReadAll(entry.Reader)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	expected := "!"
	if string(contents) != expected {
		log.Fatalf("unexpected file contents: got %v, want %v", string(contents), expected)
	}
}*/

func mustParseExpirationTime(s string) picoshare.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return picoshare.ExpirationTime(et)
}
