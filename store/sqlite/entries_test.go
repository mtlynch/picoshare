package sqlite_test

import (
	"bytes"
	"io"
	"log"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	chunkSize := uint64(5)
	dataStore := test_sqlite.NewWithChunkSize(chunkSize)

	input := "hello, world!"
	if err := dataStore.InsertEntry(bytes.NewBufferString(input), picoshare.UploadMetadata{
		ID:       picoshare.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(input)),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entryFile, err := dataStore.ReadEntryFile("dummy-id")
	if err != nil {
		t.Fatalf("failed to get entry from DB: %v", err)
	}

	contents, err := io.ReadAll(entryFile)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	if got, want := string(contents), input; got != want {
		log.Fatalf("contents=%s, want=%s", got, want)
	}

	meta, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if len(meta) != 1 {
		t.Fatalf("unexpected metadata size: got %v, want %v", len(meta), 1)
	}

	if got, want := meta[0].Size, mustParseFileSize(len(input)); !got.Equal(want) {
		t.Fatalf("unexpected file size in entry metadata: got %v, want %v", got, want)
	}

	if meta[0].DownloadCount != 0 {
		t.Fatalf("unexpected download count in entry metadata: got %v, want %v", meta[0].DownloadCount, 0)
	}

	if got, want := meta[0].Filename, picoshare.Filename("dummy-file.txt"); got != want {
		t.Fatalf("filename=%s, want=%s", got, want)
	}

	err = dataStore.DeleteEntry(picoshare.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to delete entry: %v", err)
	}

	meta, err = dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if got, want := len(meta), 0; got != want {
		t.Fatalf("metadata size=%d, want=%d", got, want)
	}
}

func TestReadLastByteOfEntry(t *testing.T) {
	chunkSize := uint64(5)
	db := test_sqlite.NewWithChunkSize(chunkSize)

	input := "hello, world!"
	if err := db.InsertEntry(bytes.NewBufferString(input), picoshare.UploadMetadata{
		ID:       picoshare.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(input)),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entryFile, err := db.ReadEntryFile(picoshare.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to read entry: %v", err)
	}

	pos, err := entryFile.Seek(1, io.SeekEnd)
	if err != nil {
		t.Fatalf("failed to seek file reader: %v", err)
	}

	expectedPos := int64(12)
	if pos != expectedPos {
		t.Fatalf("unexpected file position: got %d, want %d", pos, expectedPos)
	}

	contents, err := io.ReadAll(entryFile)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	if got, want := string(contents), "!"; got != want {
		log.Fatalf("unexpected file contents: got %v, want %v", got, want)
	}
}

func mustParseExpirationTime(s string) picoshare.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return picoshare.ExpirationTime(et)
}

func mustParseFileSize(val int) picoshare.FileSize {
	fileSize, err := picoshare.FileSizeFromInt(val)
	if err != nil {
		panic(err)
	}

	return fileSize
}
