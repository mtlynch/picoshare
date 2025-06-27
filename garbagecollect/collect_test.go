package garbagecollect_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestCollectDoesNothingWhenStoreIsEmpty(t *testing.T) {
	dataStore := test_sqlite.New()
	c := garbagecollect.NewCollector(dataStore)
	err := c.Collect()
	if err != nil {
		t.Fatalf("garbage collection failed: %v", err)
	}

	remaining, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("retrieving datastore metadata failed: %v", err)
	}

	expected := []picoshare.UploadMetadata{}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %+v, want %+v", remaining, expected)
	}
}

func TestCollectExpiredFile(t *testing.T) {
	dataStore := test_sqlite.New()
	d := "dummy data"
	expireInFiveMins := makeRelativeExpirationTime(5 * time.Minute)
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("AAAAAAAAAAAA"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("2024-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("BBBBBBBBBBBB"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("CCCCCCCCCCCC"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("DDDDDDDDDDDD"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  makeRelativeExpirationTime(-1 * time.Second),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("EEEEEEEEEEEE"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  expireInFiveMins,
			Size:     mustParseFileSize(len(d)),
		})

	c := garbagecollect.NewCollector(dataStore)
	err := c.Collect()
	if err != nil {
		t.Fatalf("garbage collection failed: %v", err)
	}

	remaining, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("retrieving datastore metadata failed: %v", err)
	}

	expected := []picoshare.UploadMetadata{
		{
			ID:       picoshare.EntryID("BBBBBBBBBBBB"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       picoshare.EntryID("CCCCCCCCCCCC"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       picoshare.EntryID("EEEEEEEEEEEE"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  expireInFiveMins,
			Size:     mustParseFileSize(len(d)),
		},
	}
	if diff := deep.Equal(expected, remaining); diff != nil {
		t.Errorf("unexpected results in datastore: got %v, want %v, diff = %v", remaining, expected, diff)
		t.Errorf("got=%+v", remaining)
		t.Errorf("want=%+v", expected)
		t.Errorf("diff=%+v", diff)
		t.FailNow()
	}
}

func TestCollectDoesNothingWhenNoFilesAreExpired(t *testing.T) {
	dataStore := test_sqlite.New()
	d := "dummy data"
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("AAAAAAAAAAAA"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("BBBBBBBBBBBB"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("CCCCCCCCCCCC"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		})

	c := garbagecollect.NewCollector(dataStore)
	err := c.Collect()
	if err != nil {
		t.Fatalf("garbage collection failed: %v", err)
	}

	remaining, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("retrieving datastore metadata failed: %v", err)
	}

	// Sort the elements so they have a consistent ordering.
	sort.Slice(remaining, func(i, j int) bool {
		return (time.Time(remaining[i].Expires)).After(time.Time(remaining[j].Expires))
	})

	expected := []picoshare.UploadMetadata{
		{
			ID:       picoshare.EntryID("AAAAAAAAAAAA"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       picoshare.EntryID("BBBBBBBBBBBB"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       picoshare.EntryID("CCCCCCCCCCCC"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		},
	}

	if diff := deep.Equal(expected, remaining); diff != nil {
		t.Errorf("unexpected results in datastore: got %v, want %v, diff = %v", remaining, expected, diff)
		t.Errorf("got=%+v", remaining)
		t.Errorf("want=%+v", expected)
		t.Errorf("diff=%+v", diff)
		t.FailNow()
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

func makeRelativeExpirationTime(delta time.Duration) picoshare.ExpirationTime {
	return picoshare.ExpirationTime(time.Now().UTC().Add(delta).Truncate(time.Second))
}

func TestCollectExpiredFileWithDownloadHistory(t *testing.T) {
	dataStore := test_sqlite.New()

	// First, test that foreign key constraints are actually working by trying
	// to insert a download record for a non-existent entry.
	nonExistentID := picoshare.EntryID("NON_EXISTENT")
	downloadRecord := picoshare.DownloadRecord{
		Time:      mustParseTime("2023-06-01T12:00:00Z"),
		ClientIP:  "192.168.1.1",
		UserAgent: "test-agent",
	}
	if err := dataStore.InsertEntryDownload(nonExistentID, downloadRecord); err == nil {
		t.Fatalf("expected foreign key constraint error when inserting download for non-existent entry, but got no error")
	}

	// Create an expired entry.
	entryID := picoshare.EntryID("EXPIRED_WITH_DOWNLOADS")
	entryData := "test file content"
	if err := dataStore.InsertEntry(strings.NewReader(entryData),
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
	if err := dataStore.InsertEntryDownload(entryID, downloadRecord); err != nil {
		t.Fatalf("failed to insert download record: %v", err)
	}

	// Attempt garbage collection. This should now succeed because we properly
	// delete download records before deleting entries.
	c := garbagecollect.NewCollector(dataStore)
	if err := c.Collect(); err != nil {
		t.Fatalf("garbage collection failed: %v", err)
	}

	// Verify that the expired entry and its download history were both deleted.
	remaining, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("retrieving datastore metadata failed: %v", err)
	}

	if len(remaining) != 0 {
		t.Fatalf("expected no entries to remain after garbage collection, but found %d entries", len(remaining))
	}

	// Verify that download history was also deleted.
	downloads, err := dataStore.GetEntryDownloads(entryID)
	if err != nil {
		t.Fatalf("failed to get entry downloads: %v", err)
	}

	if len(downloads) != 0 {
		t.Fatalf("expected no download records to remain after garbage collection, but found %d records", len(downloads))
	}
}

func mustParseFileSize(val int) picoshare.FileSize {
	fileSize, err := picoshare.FileSizeFromInt(val)
	if err != nil {
		panic(err)
	}

	return fileSize
}
