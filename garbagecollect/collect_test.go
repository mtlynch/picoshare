package garbagecollect_test

import (
	"bytes"
	"io"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestCollectDoesNothingWhenStoreIsEmpty(t *testing.T) {
	dataStore := sqlite.New("file::memory:?cache=shared")
	c := garbagecollect.NewCollector(dataStore)
	err := c.Collect()
	if err != nil {
		t.Fatalf("garbage collection failed: %v", err)
	}

	remaining, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("retrieving datastore metadata failed: %v", err)
	}

	expected := []types.UploadMetadata{}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}

func TestCollectExpiredFile(t *testing.T) {
	dataStore := sqlite.New("file::memory:?cache=shared")
	d := "dummy data"
	dataStore.InsertEntry(makeData(d),
		types.UploadMetadata{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("2000-01-01T00:00:00Z"),
			Size:    len(d),
		})
	dataStore.InsertEntry(makeData(d),
		types.UploadMetadata{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    len(d),
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

	expected := []types.UploadMetadata{
		{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    len(d),
		},
	}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}

func TestCollectDoesNothingWhenNoFilesAreExpired(t *testing.T) {
	dataStore := sqlite.New("file::memory:?cache=shared")
	d := "dummy data"
	dataStore.InsertEntry(makeData(d),
		types.UploadMetadata{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:    len(d),
		})
	dataStore.InsertEntry(makeData(d),
		types.UploadMetadata{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    len(d),
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

	sort.Slice(remaining, func(i, j int) bool {
		return (time.Time(remaining[i].Expires)).After(time.Time(remaining[j].Expires))
	})

	expected := []types.UploadMetadata{
		{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:    len(d),
		},
		{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    len(d),
		},
	}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}

func makeData(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}

func mustParseExpirationTime(s string) types.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}
