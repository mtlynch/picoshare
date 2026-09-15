package sqlite_test

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	chunkSize := uint64(5)
	dataStore := test_sqlite.NewWithChunkSize(t, chunkSize)
	entryID := mustCreateEntryID(t, "abcdefghij")

	input := "hello, world!"
	if err := dataStore.InsertEntry(bytes.NewBufferString(input), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "dummy-file.txt",
		Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(input)),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entryFile, err := dataStore.ReadEntryFile(entryID)
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

	err = dataStore.DeleteEntry(entryID)
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
	db := test_sqlite.NewWithChunkSize(t, chunkSize)
	entryID := mustCreateEntryID(t, "abcdefghij")

	input := "hello, world!"
	if err := db.InsertEntry(bytes.NewBufferString(input), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "dummy-file.txt",
		Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(input)),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entryFile, err := db.ReadEntryFile(entryID)
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

func TestUpdateEntryMetadata(t *testing.T) {
	dataStore := test_sqlite.New(t)
	entryID := mustCreateEntryID(t, "abcdefghij")
	passphrase, err := picoshare.NewDownloadPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create download passphrase: %v", err)
	}

	data := "dummy data"
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:                 entryID,
		Filename:           "dummy-file.txt",
		Uploaded:           mustParseTime("2025-05-25T00:00:00Z"),
		Expires:            mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:               mustParseFileSize(len(data)),
		DownloadPassphrase: passphrase,
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	metadata, err := dataStore.GetEntryMetadata(entryID)
	if err != nil {
		t.Fatalf("failed to retrieve entry metadata: %v", err)
	}
	if got, want := metadata.DownloadPassphrase.String(), "correct horse battery staple"; got != want {
		t.Errorf("download passphrase=%q, want=%q", got, want)
	}

	if err := dataStore.UpdateEntryMetadata(entryID, picoshare.UploadMetadata{
		Filename:           "renamed-file.txt",
		Expires:            mustParseExpirationTime("2041-01-01T00:00:00Z"),
		Note:               picoshare.FileNote{Value: new("updated note")},
		DownloadPassphrase: picoshare.DownloadPassphrase{},
	}); err != nil {
		t.Fatalf("failed to update entry metadata: %v", err)
	}

	metadata, err = dataStore.GetEntryMetadata(entryID)
	if err != nil {
		t.Fatalf("failed to retrieve entry metadata: %v", err)
	}
	if got, want := metadata.Filename, picoshare.Filename("renamed-file.txt"); got != want {
		t.Errorf("filename=%q, want=%q", got, want)
	}
	if got, want := metadata.Expires, mustParseExpirationTime("2041-01-01T00:00:00Z"); got != want {
		t.Errorf("expiration=%v, want=%v", got, want)
	}
	if got, want := metadata.Note.String(), "updated note"; got != want {
		t.Errorf("note=%q, want=%q", got, want)
	}
	if !metadata.DownloadPassphrase.Empty() {
		t.Errorf("download passphrase=%q, want empty", metadata.DownloadPassphrase.String())
	}
}

// File listing, garbage collection, and database-size checks read every
// entry's metadata but never need download passphrases, so the bulk read
// leaves them out.
func TestGetEntriesMetadataOmitsDownloadPassphrase(t *testing.T) {
	dataStore := test_sqlite.New(t)
	entryID := mustCreateEntryID(t, "abcdefghij")
	passphrase, err := picoshare.NewDownloadPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create download passphrase: %v", err)
	}

	data := "dummy data"
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:                 entryID,
		Filename:           "dummy-file.txt",
		Uploaded:           mustParseTime("2025-05-25T00:00:00Z"),
		Expires:            mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:               mustParseFileSize(len(data)),
		DownloadPassphrase: passphrase,
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entries, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to retrieve entries metadata: %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("entries count=%d, want=%d", got, want)
	}
	if !entries[0].DownloadPassphrase.Empty() {
		t.Errorf("download passphrase=%q, want empty", entries[0].DownloadPassphrase.String())
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
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

func mustCreateEntryID(t *testing.T, raw string) picoshare.EntryID {
	t.Helper()

	id, err := picoshare.NewEntryID(raw)
	if err != nil {
		t.Fatalf("failed to create entry ID %q: %v", raw, err)
	}
	return id
}
