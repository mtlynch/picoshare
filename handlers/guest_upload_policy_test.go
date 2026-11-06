package handlers_test

import (
	"errors"
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

type failingUploadStore struct {
	handlers.Store
	err error
}

func (s failingUploadStore) InsertEntry(io.Reader, picoshare.UploadMetadata) error {
	return s.err
}

func (r *beforeReadReader) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		beforeRead := r.beforeRead
		r.beforeRead = nil
		beforeRead()
	}
	return r.Reader.Read(p)
}

func TestGuestUploadRevokedWhileBodyWithheld(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		method      string
		path        string
		status      int
	}{
		{
			explanation: "disabling a link before receiving the body rejects the held upload",
			method:      http.MethodPut,
			path:        "/api/guest-links/abcdefgh23456789/disable",
			status:      http.StatusNoContent,
		},
		{
			explanation: "deleting a link before receiving the body rejects the held upload",
			method:      http.MethodDelete,
			path:        "/api/guest-links/abcdefgh23456789",
			status:      http.StatusOK,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := mustParseTime("2025-01-01T00:00:00Z")
			dataStore := test_sqlite.NewWithClock(t, func() time.Time { return now })
			if err := dataStore.InsertGuestLink(picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			}); err != nil {
				t.Fatalf("failed to insert guest link: %v", err)
			}
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
			body, contentType := createMultipartFormBody("held.txt", "", "", strings.NewReader("held payload"))
			heldBody := &beforeReadReader{
				Reader: body,
				beforeRead: func() {
					req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, tt.status; got != want {
						t.Fatalf("policy change status=%d, want=%d: %s", got, want, rec.Body.String())
					}
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

			entries, err := dataStore.GetEntriesMetadata()
			if err != nil {
				t.Fatalf("failed to list entries: %v", err)
			}
			if got, want := len(entries), 0; got != want {
				t.Errorf("stored entries=%d, want=%d", got, want)
			}
		})
	}
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

func TestUploadPersistenceErrors(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		path        string
		err         error
		status      int
		message     string
	}{
		{
			explanation: "a guest policy rejection returns unauthorized",
			path:        "/api/guest/abcdefgh23456789",
			err:         store.GuestUploadRejectedError{ID: "abcdefgh23456789"},
			status:      http.StatusUnauthorized,
			message:     "Guest link no longer permits this upload",
		},
		{
			explanation: "a guest database failure returns a server error with its cause",
			path:        "/api/guest/abcdefgh23456789",
			err:         errors.New("failed to remove upload chunks"),
			status:      http.StatusInternalServerError,
			message:     "Failed to save uploaded file: database error: failed to remove upload chunks",
		},
		{
			explanation: "an owner database failure returns a server error with its cause",
			path:        "/api/entry?expiration=2030-01-01T00:00:00Z",
			err:         errors.New("failed to insert entry metadata"),
			status:      http.StatusInternalServerError,
			message:     "Failed to save uploaded file: database error: failed to insert entry metadata",
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := mustParseTime("2025-01-01T00:00:00Z")
			dataStore := test_sqlite.NewWithClock(t, func() time.Time { return now })
			if err := dataStore.InsertGuestLink(picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			}); err != nil {
				t.Fatalf("failed to insert guest link: %v", err)
			}
			failingStore := failingUploadStore{Store: &dataStore, err: tt.err}
			s := handlers.New(mockAuthenticator{}, failingStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
			body, contentType := createMultipartFormBody("upload.txt", "", "", strings.NewReader("payload"))
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, tt.status; got != want {
				t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
			}
			if got, want := strings.TrimSpace(rec.Body.String()), tt.message; got != want {
				t.Errorf("response=%q, want=%q", got, want)
			}
		})
	}
}

func TestGuestLinkIndexUsesCurrentTime(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		now         time.Time
		wantLink    bool
	}{
		{
			explanation: "an unexpired link is clickable using the injected clock",
			now:         mustParseTime("2025-01-01T00:00:00Z"),
			wantLink:    true,
		},
		{
			explanation: "an expired link is listed but is not clickable",
			now:         mustParseTime("2025-01-02T00:00:00Z"),
			wantLink:    false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.NewWithClock(t, func() time.Time { return tt.now })
			if err := dataStore.InsertGuestLink(picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Label:           "Guest uploads",
				Created:         mustParseTime("2024-12-31T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2025-01-02T00:00:00Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			}); err != nil {
				t.Fatalf("failed to insert guest link: %v", err)
			}
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return tt.now })
			req := httptest.NewRequest(http.MethodGet, "/guest-links", http.NoBody)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("index status=%d, want=%d: %s", got, want, rec.Body.String())
			}
			if got, want := strings.Contains(rec.Body.String(), "Guest uploads"), true; got != want {
				t.Errorf("index contains guest link label=%v, want=%v", got, want)
			}
			if got, want := strings.Contains(rec.Body.String(), `href="/g/abcdefgh23456789"`), tt.wantLink; got != want {
				t.Errorf("index contains clickable guest link=%v, want=%v", got, want)
			}
		})
	}
}
