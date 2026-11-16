package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type beforeGuestUploadRead struct {
	io.Reader
	beforeRead func()
}

func (r *beforeGuestUploadRead) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		beforeRead := r.beforeRead
		r.beforeRead = nil
		beforeRead()
	}
	return r.Reader.Read(p)
}

func TestServerRejectsParallelLimitedGuestUploads(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		uploadLimit   int
		firstNote     string
		firstStatus   int
		laterStatus   int
		storedUploads int
	}{
		{
			explanation:   "the server rejects parallel uploads even when multiple slots remain",
			uploadLimit:   2,
			firstStatus:   http.StatusOK,
			laterStatus:   http.StatusOK,
			storedUploads: 2,
		},
		{
			explanation:   "the server releases access after an invalid upload without consuming a slot",
			uploadLimit:   1,
			firstNote:     "guest uploads cannot contain notes",
			firstStatus:   http.StatusBadRequest,
			laterStatus:   http.StatusOK,
			storedUploads: 1,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			// Open SQLite outside the bubble so its connection-maintenance
			// goroutines do not outlive the test bubble.
			dataStore := test_sqlite.New(t)
			now := mustParseTime("2025-01-01T00:00:00Z")
			mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileUploads:  new(tt.uploadLimit),
			})
			synctest.Test(t, func(t *testing.T) {
				s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
				release := make(chan struct{})
				releaseFirst := sync.OnceFunc(func() { close(release) })
				defer func() { releaseFirst(); synctest.Wait() }()
				firstStarted := make(chan struct{})
				first := httptest.NewRecorder()
				{
					body, contentType := createMultipartFormBody("first.txt", tt.firstNote, "", strings.NewReader("first payload"))
					heldBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() {
						close(firstStarted)
						<-release
					}}
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", heldBody)
					req.Header.Set("Content-Type", contentType)
					go s.Router().ServeHTTP(first, req)
				}
				<-firstStarted
				{
					bodyRead := false
					body, contentType := createMultipartFormBody("parallel.txt", "", "", strings.NewReader("parallel payload"))
					observedBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() { bodyRead = true }}
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", observedBody)
					req.Header.Set("Content-Type", contentType)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, http.StatusConflict; got != want {
						t.Fatalf("parallel upload status=%d, want=%d: %s", got, want, rec.Body.String())
					}
					if got, want := rec.Body.String(), "This guest link does not allow you to perform multiple uploads at once\n"; got != want {
						t.Errorf("rejection=%q, want=%q", got, want)
					}
					if got, want := bodyRead, false; got != want {
						t.Errorf("server read the rejected request body=%v, want=%v", got, want)
					}
				}
				releaseFirst()
				synctest.Wait()
				if got, want := first.Code, tt.firstStatus; got != want {
					t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
				}
				{
					body, contentType := createMultipartFormBody("later.txt", "", "", strings.NewReader("later payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, tt.laterStatus; got != want {
						t.Fatalf("later status=%d, want=%d: %s", got, want, rec.Body.String())
					}
				}
				entries, err := dataStore.GetEntriesMetadata()
				if err != nil {
					t.Fatalf("failed to list uploads: %v", err)
				}
				if got, want := len(entries), tt.storedUploads; got != want {
					t.Errorf("stored uploads=%d, want=%d", got, want)
				}
			})
		})
	}
}

func mustInsertGuestUploadLink(t *testing.T, s handlers.Store, link picoshare.GuestLink) {
	t.Helper()
	if err := s.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
}
