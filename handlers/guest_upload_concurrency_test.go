package handlers_test

import (
	"errors"
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

type guestUploadLookupStore struct {
	handlers.Store
	lookup func(picoshare.GuestLinkID) (picoshare.GuestLink, error)
}

type heldGuestUploadStore struct {
	handlers.Store
	beforeInsert func(picoshare.UploadMetadata) error
}

func (r *beforeGuestUploadRead) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		beforeRead := r.beforeRead
		r.beforeRead = nil
		beforeRead()
	}
	return r.Reader.Read(p)
}

func (s guestUploadLookupStore) GetGuestLink(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
	return s.lookup(id)
}

func (s heldGuestUploadStore) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	if err := s.beforeInsert(metadata); err != nil {
		return err
	}
	return s.Store.InsertEntry(reader, metadata)
}

func TestLimitedGuestUploadsRejectParallelRequests(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		uploadLimit   int
		firstNote     string
		firstStatus   int
		laterStatus   int
		storedUploads int
	}{
		{
			explanation:   "parallel requests are forbidden even when multiple slots remain",
			uploadLimit:   2,
			firstStatus:   http.StatusOK,
			laterStatus:   http.StatusOK,
			storedUploads: 2,
		},
		{
			explanation:   "parallel requests conflict but a later request sees the exhausted link",
			uploadLimit:   1,
			firstStatus:   http.StatusOK,
			laterStatus:   http.StatusUnauthorized,
			storedUploads: 1,
		},
		{
			explanation:   "an invalid upload releases access without consuming its slot",
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
					if got, want := rec.Body.String(), "Parallel uploads are forbidden on this guest link\n"; got != want {
						t.Errorf("rejection=%q, want=%q", got, want)
					}
					if got, want := rec.Header().Get("Connection"), "close"; got != want {
						t.Errorf("connection header=%q, want=%q", got, want)
					}
					if got, want := bodyRead, false; got != want {
						t.Errorf("rejected request read its body=%v, want=%v", got, want)
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

func TestUnlimitedGuestUploadsRemainConcurrent(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	})
	synctest.Test(t, func(t *testing.T) {
		s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
		release := make(chan struct{})
		releaseFirst := sync.OnceFunc(func() { close(release) })
		defer func() { releaseFirst(); synctest.Wait() }()
		firstStarted := make(chan struct{})
		first := httptest.NewRecorder()
		{
			body, contentType := createMultipartFormBody("first.txt", "", "", strings.NewReader("first payload"))
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
			body, contentType := createMultipartFormBody("second.txt", "", "", strings.NewReader("second payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("concurrent upload status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		}
		releaseFirst()
		synctest.Wait()
		if got, want := first.Code, http.StatusOK; got != want {
			t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
		}
	})
}

func TestLimitedGuestUploadHoldsClaimThroughPersistence(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		insertError error
		firstStatus int
		laterStatus int
	}{
		{
			explanation: "persistence retains exclusive access until the last slot is consumed",
			firstStatus: http.StatusOK,
			laterStatus: http.StatusUnauthorized,
		},
		{
			explanation: "a persistence failure releases access without consuming the slot",
			insertError: errors.New("failed to persist upload"),
			firstStatus: http.StatusInternalServerError,
			laterStatus: http.StatusOK,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			now := mustParseTime("2025-01-01T00:00:00Z")
			mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileUploads:  new(1),
			})
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				releaseFirst := sync.OnceFunc(func() { close(release) })
				defer func() { releaseFirst(); synctest.Wait() }()
				firstInserting := make(chan struct{})
				heldStore := heldGuestUploadStore{Store: &dataStore, beforeInsert: func(metadata picoshare.UploadMetadata) error {
					if metadata.Filename == "first.txt" {
						close(firstInserting)
						<-release
						return tt.insertError
					}
					return nil
				}}
				s := handlers.New(mockAuthenticator{}, heldStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
				first := httptest.NewRecorder()
				{
					body, contentType := createMultipartFormBody("first.txt", "", "", strings.NewReader("first payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					go s.Router().ServeHTTP(first, req)
				}
				<-firstInserting
				{
					body, contentType := createMultipartFormBody("parallel.txt", "", "", strings.NewReader("parallel payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, http.StatusConflict; got != want {
						t.Fatalf("upload during persistence status=%d, want=%d: %s", got, want, rec.Body.String())
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
						t.Fatalf("later upload status=%d, want=%d: %s", got, want, rec.Body.String())
					}
				}
				entries, err := dataStore.GetEntriesMetadata()
				if err != nil {
					t.Fatalf("failed to list uploads: %v", err)
				}
				if got, want := len(entries), 1; got != want {
					t.Errorf("stored uploads=%d, want=%d", got, want)
				}
			})
		})
	}
}

func TestLimitedGuestUploadsOnDifferentLinksRemainConcurrent(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	})
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "bcdefghj23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	})
	synctest.Test(t, func(t *testing.T) {
		s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
		release := make(chan struct{})
		releaseFirst := sync.OnceFunc(func() { close(release) })
		defer func() { releaseFirst(); synctest.Wait() }()
		firstStarted := make(chan struct{})
		first := httptest.NewRecorder()
		{
			body, contentType := createMultipartFormBody("first.txt", "", "", strings.NewReader("first payload"))
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
			body, contentType := createMultipartFormBody("second.txt", "", "", strings.NewReader("second payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/bcdefghj23456789", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("other link upload status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		}
		releaseFirst()
		synctest.Wait()
		if got, want := first.Code, http.StatusOK; got != want {
			t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
		}
	})
}

func TestGuestUploadRefreshesStaleCountAfterClaim(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	})
	synctest.Test(t, func(t *testing.T) {
		snapshotLoaded := make(chan struct{})
		release := make(chan struct{})
		releaseSnapshot := sync.OnceFunc(func() { close(release) })
		defer func() { releaseSnapshot(); synctest.Wait() }()
		holdSnapshot := true
		lookupStore := guestUploadLookupStore{Store: &dataStore, lookup: func(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
			gl, err := dataStore.GetGuestLink(id)
			if holdSnapshot {
				holdSnapshot = false
				close(snapshotLoaded)
				<-release
			}
			return gl, err
		}}
		s := handlers.New(mockAuthenticator{}, lookupStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
		stale := httptest.NewRecorder()
		{
			body, contentType := createMultipartFormBody("stale.txt", "", "", strings.NewReader("stale payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			go s.Router().ServeHTTP(stale, req)
		}
		<-snapshotLoaded
		// Consume the slot after the other request read its initial snapshot,
		// but before it claims access to the link.
		{
			body, contentType := createMultipartFormBody("winner.txt", "", "", strings.NewReader("winning payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("winning status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		}
		releaseSnapshot()
		synctest.Wait()
		if got, want := stale.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("stale request status=%d, want=%d: %s", got, want, stale.Body.String())
		}
		entries, err := dataStore.GetEntriesMetadata()
		if err != nil {
			t.Fatalf("failed to list uploads: %v", err)
		}
		if got, want := len(entries), 1; got != want {
			t.Errorf("stored uploads=%d, want=%d", got, want)
		}
	})
}

func TestGuestUploadReleasesClaimAfterLookupFailure(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	})
	synctest.Test(t, func(t *testing.T) {
		snapshotLoaded := make(chan struct{})
		release := make(chan struct{})
		releaseSnapshot := sync.OnceFunc(func() { close(release) })
		defer func() { releaseSnapshot(); synctest.Wait() }()
		holdSnapshot := true
		var lookupError error
		lookupStore := guestUploadLookupStore{Store: &dataStore, lookup: func(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
			if lookupError != nil {
				return picoshare.GuestLink{}, lookupError
			}
			gl, err := dataStore.GetGuestLink(id)
			if holdSnapshot {
				holdSnapshot = false
				close(snapshotLoaded)
				<-release
			}
			return gl, err
		}}
		s := handlers.New(mockAuthenticator{}, lookupStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
		failed := httptest.NewRecorder()
		{
			body, contentType := createMultipartFormBody("failed.txt", "", "", strings.NewReader("failed payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			go s.Router().ServeHTTP(failed, req)
		}
		<-snapshotLoaded
		synctest.Wait()
		lookupError = errors.New("guest link lookup failed")
		releaseSnapshot()
		synctest.Wait()
		if got, want := failed.Code, http.StatusInternalServerError; got != want {
			t.Fatalf("failed lookup status=%d, want=%d: %s", got, want, failed.Body.String())
		}
		lookupError = nil
		{
			body, contentType := createMultipartFormBody("later.txt", "", "", strings.NewReader("later payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("later status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		}
	})
}

func mustInsertGuestUploadLink(t *testing.T, s handlers.Store, link picoshare.GuestLink) {
	t.Helper()
	if err := s.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
}
