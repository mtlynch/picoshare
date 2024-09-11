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
			ID:      picoshare.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("2000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("DDDDDDDDDDDD"),
			Expires: makeRelativeExpirationTime(-1 * time.Second),
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("EEEEEEEEEEEE"),
			Expires: expireInFiveMins,
			Size:    uint64(len(d)),
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
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		},
		{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
			Size:    uint64(len(d)),
		},
		{
			ID:      picoshare.EntryID("EEEEEEEEEEEE"),
			Expires: expireInFiveMins,
			Size:    uint64(len(d)),
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
			ID:      picoshare.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
			Size:    uint64(len(d)),
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
			ID:      picoshare.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		},
		{
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
			Size:    uint64(len(d)),
		},
		{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
			Size:    uint64(len(d)),
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
