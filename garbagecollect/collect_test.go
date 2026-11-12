package garbagecollect_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/internal/testutil"
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
	d := "dummy data"
	expireInFiveMins := picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "2025-01-01T00:05:00Z"))
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "AAAAAAAAAA"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "2024-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntryDownload(
		testutil.MustCreate(t, picoshare.EntryIDFromString, "AAAAAAAAAA"),
		picoshare.DownloadRecord{
			Time:      testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-06-01T12:00:00Z"),
			ClientIP:  "192.168.1.1",
			UserAgent: "test-agent",
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "BBBBBBBBBB"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "3000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "CCCCCCCCCC"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "DDDDDDDDDD"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "2024-12-31T23:59:59Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "EEEEEEEEEE"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  expireInFiveMins,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
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
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "BBBBBBBBBB"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "3000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		},
		{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "CCCCCCCCCC"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		},
		{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "EEEEEEEEEE"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  expireInFiveMins,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
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
	d := "dummy data"
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "AAAAAAAAAA"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "4000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "BBBBBBBBBB"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "3000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "CCCCCCCCCC"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
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
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "AAAAAAAAAA"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "4000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		},
		{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "BBBBBBBBBB"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "3000-01-01T00:00:00Z")),
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
		},
		{
			ID:       testutil.MustCreate(t, picoshare.EntryIDFromString, "CCCCCCCCCC"),
			Uploaded: testutil.MustCreate2(t, time.Parse, time.RFC3339, "2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     testutil.MustCreate(t, picoshare.FileSizeFromInt, len(d)),
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
