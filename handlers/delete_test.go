package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

var nilSpaceChecker handlers.SpaceChecker
var nilGarbageCollector *garbagecollect.Collector

func TestDeleteExistingFile(t *testing.T) {
	dataStore := test_sqlite.New()
	fileContents := "dummy data"
	dataStore.InsertEntry(strings.NewReader(fileContents),
		picoshare.UploadMetadata{
			ID:       picoshare.EntryID("hR87apiUCj"),
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  mustParseExpirationTime("2024-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(fileContents)),
		})
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCj", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if status := res.StatusCode; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	_, err = dataStore.GetEntryMetadata(picoshare.EntryID("hR87apiUCj"))
	if _, ok := err.(store.EntryNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted", picoshare.EntryID("hR87apiUCj"))
	}
}

func TestDeleteNonExistentFile(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCj", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := res.StatusCode; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestDeleteInvalidEntryID(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/entry/invalid-entry-id", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := res.StatusCode; status != http.StatusBadRequest {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}
