package handlers_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type beforeReadReader struct {
	io.Reader
	beforeRead func()
}

func (r *beforeReadReader) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		beforeRead := r.beforeRead
		r.beforeRead = nil
		beforeRead()
	}
	return r.Reader.Read(p)
}

func TestGuestUploadExpiresWhileBodyWithheld(t *testing.T) {
	now := mustParseTime("2025-01-01T00:00:00Z")
	dataStore := test_sqlite.NewWithClock(t, func() time.Time { return now })
	if err := dataStore.InsertGuestLink(picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      mustParseExpirationTime("2025-01-01T01:00:00Z"),
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	}); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	body, contentType := createMultipartFormBody("held.txt", "", "", strings.NewReader("held payload"))
	bodyRead := false
	heldBody := &beforeReadReader{
		Reader: body,
		beforeRead: func() {
			bodyRead = true
			now = mustParseTime("2025-01-01T01:00:01Z")
		},
	}

	{
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", heldBody)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
		}
	}
	if got, want := bodyRead, true; got != want {
		t.Errorf("request reached body read=%v, want=%v", got, want)
	}
	entries, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if got, want := len(entries), 0; got != want {
		t.Errorf("stored entries=%d, want=%d", got, want)
	}
}

func TestGuestUploadPostPolicyRejection(t *testing.T) {
	now := mustParseTime("2025-01-01T00:00:00Z")
	dataStore := test_sqlite.New(t)
	guestLinkID := picoshare.GuestLinkID("abcdefgh23456789")
	if err := dataStore.InsertGuestLink(picoshare.GuestLink{
		ID:              guestLinkID,
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	}); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	failingStore := failingUploadStore{
		Store:            &dataStore,
		insertEntryError: store.ErrGuestUploadRejected,
	}
	s := handlers.New(mockAuthenticator{}, failingStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	body, contentType := createMultipartFormBody("upload.txt", "", "", strings.NewReader("payload"))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/guest/%s", guestLinkID), body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.Router().ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusUnauthorized; got != want {
		t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
	}
	if got, want := strings.TrimSpace(rec.Body.String()), "Guest link no longer permits this upload"; got != want {
		t.Errorf("response=%q, want=%q", got, want)
	}
}
