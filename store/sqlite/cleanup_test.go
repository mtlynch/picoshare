package sqlite_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestDeleteExpiredEntriesWithDownloadHistory(t *testing.T) {
	dataStore := test_sqlite.New()

	// Create an expired entry.
	entryID := picoshare.EntryID("EXPIRED_ENTRY")
	entryData := "test file content"
	if err := dataStore.InsertEntry(bytes.NewBufferString(entryData),
		picoshare.UploadMetadata{
			ID:       entryID,
			Filename: "test.txt",
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("2022-03-01T00:00:00Z"), // Expired.
			Size:     mustParseFileSize(len(entryData)),
		}); err != nil {
		t.Fatalf("failed to insert entry: %v", err)
	}

	// Record a download for the expired entry.
	downloadRecord := picoshare.DownloadRecord{
		Time:      mustParseTime("2023-06-01T12:00:00Z"),
		ClientIP:  "192.168.1.1",
		UserAgent: "test-agent",
	}
	if err := dataStore.InsertEntryDownload(entryID, downloadRecord); err != nil {
		t.Fatalf("failed to insert download record: %v", err)
	}

	// Attempt to purge expired entries. This should fail due to foreign key
	// constraint violation because the downloads table still references the
	// entry we're trying to delete.
	if got, want := dataStore.Purge() != nil, true; got != want {
		t.Fatalf("expected Purge() to fail with foreign key constraint error, but it succeeded")
	}

	// Verify the error is related to foreign key constraints.
	err := dataStore.Purge()
	if got, want := strings.Contains(err.Error(), "FOREIGN KEY constraint failed"), true; got != want {
		t.Fatalf("expected foreign key constraint error, got: %v", err)
	}
}
