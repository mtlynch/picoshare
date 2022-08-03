package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

var nilGarbageCollector *garbagecollect.Collector

func TestDeleteExistingFile(t *testing.T) {
	dataStore := test_sqlite.New()
	dataStore.InsertEntry(strings.NewReader("dummy data"),
		types.UploadMetadata{
			ID: types.EntryID("hR87apiUCj"),
		})

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCj", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	_, err = dataStore.GetEntry(types.EntryID("hR87apiUCj"))
	if _, ok := err.(store.EntryNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted", types.EntryID("hR87apiUCj"))
	}
}

func TestDeleteNonExistentFile(t *testing.T) {
	dataStore := test_sqlite.New()

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCj", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestDeleteInvalidEntryID(t *testing.T) {
	dataStore := test_sqlite.New()

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/entry/invalid-entry-id", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusBadRequest {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}
