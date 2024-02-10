package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func makeGuestUploadMaxFileBytes(i uint64) picoshare.GuestUploadMaxFileBytes {
	return picoshare.GuestUploadMaxFileBytes(&i)
}

func makeGuestUploadCountLimit(i int) picoshare.GuestUploadCountLimit {
	return picoshare.GuestUploadCountLimit(&i)
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	dataStore.InsertGuestLink(picoshare.GuestLink{
		ID:      picoshare.GuestLinkID("abcdefgh23456789"),
		Created: time.Now(),
		Expires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})

	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	_, err = dataStore.GetGuestLink(picoshare.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", picoshare.EntryID("abcdefgh23456789"), err)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestDeleteInvalidGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/guest-links/i-am-an-invalid-link", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusBadRequest {
		t.Fatalf("DELETE returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
