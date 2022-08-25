package garbagecollect_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestCollectDoesNothingWhenStoreIsEmpty(t *testing.T) {
	dataStore := test_sqlite.New()
	c := garbagecollect.NewCollector(dataStore, false)
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
	dataStore := test_sqlite.New()
	d := "dummy data"
	expireInFiveMins := makeRelativeExpirationTime(5 * time.Minute)
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("2000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("CCCCCCCCCCCC"),
			Expires: types.NeverExpire,
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("DDDDDDDDDDDD"),
			Expires: makeRelativeExpirationTime(-1 * time.Second),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("EEEEEEEEEEEE"),
			Expires: expireInFiveMins,
		})

	c := garbagecollect.NewCollector(dataStore, false)
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
			Size:    int64(len(d)),
		},
		{
			ID:      types.EntryID("CCCCCCCCCCCC"),
			Expires: types.NeverExpire,
			Size:    int64(len(d)),
		},
		{
			ID:      types.EntryID("EEEEEEEEEEEE"),
			Expires: expireInFiveMins,
			Size:    int64(len(d)),
		},
	}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}

func TestCollectDoesNothingWhenNoFilesAreExpired(t *testing.T) {
	dataStore := test_sqlite.New()
	d := "dummy data"
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		types.UploadMetadata{
			ID:      types.EntryID("CCCCCCCCCCCC"),
			Expires: types.NeverExpire,
		})

	c := garbagecollect.NewCollector(dataStore, false)
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

	expected := []types.UploadMetadata{
		{
			ID:      types.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:    int64(len(d)),
		},
		{
			ID:      types.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    int64(len(d)),
		},
		{
			ID:      types.EntryID("CCCCCCCCCCCC"),
			Expires: types.NeverExpire,
			Size:    int64(len(d)),
		},
	}
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}

func mustParseExpirationTime(s string) types.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}

func makeRelativeExpirationTime(delta time.Duration) types.ExpirationTime {
	return types.ExpirationTime(time.Now().UTC().Add(delta).Truncate(time.Second))
}
