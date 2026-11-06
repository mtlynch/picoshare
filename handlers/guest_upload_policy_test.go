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
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type failingUploadStore struct {
	handlers.Store
	err error
}

func (s failingUploadStore) InsertEntry(io.Reader, picoshare.UploadMetadata) error {
	return s.err
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
