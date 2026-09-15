package garbagecollect_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestCollectDoesNothingWhenStoreIsEmpty(t *testing.T) {
	dataStore := test_sqlite.New(t)
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
	dataStore := test_sqlite.New(t)
	aID := mustCreateEntryID(t, "AAAAAAAAAA")
	bID := mustCreateEntryID(t, "BBBBBBBBBB")
	cID := mustCreateEntryID(t, "CCCCCCCCCC")
	dID := mustCreateEntryID(t, "DDDDDDDDDD")
	eID := mustCreateEntryID(t, "EEEEEEEEEE")
	d := "dummy data"
	expireInFiveMins := mustParseExpirationTime("2025-01-01T00:05:00Z")
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       aID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("2024-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntryDownload(
		aID,
		picoshare.DownloadRecord{
			Time:      mustParseTime("2023-06-01T12:00:00Z"),
			ClientIP:  "192.168.1.1",
			UserAgent: "test-agent",
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       bID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       cID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       dID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("2024-12-31T23:59:59Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       eID,
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
			ID:       bID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       cID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       eID,
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
	dataStore := test_sqlite.New(t)
	aID := mustCreateEntryID(t, "AAAAAAAAAA")
	bID := mustCreateEntryID(t, "BBBBBBBBBB")
	cID := mustCreateEntryID(t, "CCCCCCCCCC")
	d := "dummy data"
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       aID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       bID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       cID,
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
			ID:       aID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       bID,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(d)),
		},
		{
			ID:       cID,
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
