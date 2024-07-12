package garbagecollect_test

/*func TestCollectDoesNothingWhenStoreIsEmpty(t *testing.T) {
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
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}*/

/*func TestCollectExpiredFile(t *testing.T) {
	dataStore := test_sqlite.New()
	d := "dummy data"
	expireInFiveMins := makeRelativeExpirationTime(5 * time.Minute)
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("2000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("DDDDDDDDDDDD"),
			Expires: makeRelativeExpirationTime(-1 * time.Second),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("EEEEEEEEEEEE"),
			Expires: expireInFiveMins,
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
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}*/

/*func TestCollectDoesNothingWhenNoFilesAreExpired(t *testing.T) {
	dataStore := test_sqlite.New()
	d := "dummy data"
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("AAAAAAAAAAAA"),
			Expires: mustParseExpirationTime("4000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("BBBBBBBBBBBB"),
			Expires: mustParseExpirationTime("3000-01-01T00:00:00Z"),
		})
	dataStore.InsertEntry(strings.NewReader(d),
		picoshare.UploadMetadata{
			ID:      picoshare.EntryID("CCCCCCCCCCCC"),
			Expires: picoshare.NeverExpire,
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
	if !reflect.DeepEqual(expected, remaining) {
		t.Fatalf("unexpected results in datastore: got %v, want %v", remaining, expected)
	}
}*/
/*
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
*/
