package handlers_test

import (
	"context"
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

func (r *beforeGuestUploadRead) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		beforeRead := r.beforeRead
		r.beforeRead = nil
		beforeRead()
	}
	return r.Reader.Read(p)
}

type guestUploadLookupStore struct {
	handlers.Store
	lookup func(picoshare.GuestLinkID) (picoshare.GuestLink, error)
}

func (s guestUploadLookupStore) GetGuestLink(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
	return s.lookup(id)
}

type heldGuestUploadStore struct {
	handlers.Store
	beforeInsert func(picoshare.UploadMetadata) error
}

func (s heldGuestUploadStore) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	if err := s.beforeInsert(metadata); err != nil {
		return err
	}
	return s.Store.InsertEntry(reader, metadata)
}

func TestLimitedGuestUploadsWaitForAvailableSlots(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		uploadLimit   int
		firstNote     string
		firstStatus   int
		secondStatus  int
		storedUploads int
	}{
		{
			explanation:   "two available slots allow both requests to succeed serially",
			uploadLimit:   2,
			firstStatus:   http.StatusOK,
			secondStatus:  http.StatusOK,
			storedUploads: 2,
		},
		{
			explanation:   "one available slot rejects the waiting request only after the first succeeds",
			uploadLimit:   1,
			firstStatus:   http.StatusOK,
			secondStatus:  http.StatusUnauthorized,
			storedUploads: 1,
		},
		{
			explanation:   "an invalid first upload leaves the slot available to the waiting request",
			uploadLimit:   1,
			firstNote:     "guest uploads cannot contain notes",
			firstStatus:   http.StatusBadRequest,
			secondStatus:  http.StatusOK,
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

				secondRead := false
				secondDone := false
				second := httptest.NewRecorder()
				{
					body, contentType := createMultipartFormBody("second.txt", "", "", strings.NewReader("second payload"))
					observedBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() { secondRead = true }}
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", observedBody)
					req.Header.Set("Content-Type", contentType)
					go func() {
						s.Router().ServeHTTP(second, req)
						secondDone = true
					}()
				}
				synctest.Wait()
				if got, want := secondRead, false; got != want {
					t.Errorf("waiting request read its body=%v, want=%v", got, want)
				}
				if got, want := secondDone, false; got != want {
					t.Errorf("waiting request finished=%v, want=%v", got, want)
				}

				releaseFirst()
				synctest.Wait()
				if got, want := first.Code, tt.firstStatus; got != want {
					t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
				}
				if got, want := second.Code, tt.secondStatus; got != want {
					t.Fatalf("second status=%d, want=%d: %s", got, want, second.Body.String())
				}
				if tt.secondStatus == http.StatusUnauthorized {
					if got, want := second.Body.String(), "Guest link is no longer active\n"; got != want {
						t.Errorf("rejection=%q, want=%q", got, want)
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

func TestLimitedGuestUploadHoldsLockThroughPersistence(t *testing.T) {
	for _, tt := range []struct {
		explanation  string
		insertError  error
		firstStatus  int
		secondStatus int
	}{
		{
			explanation:  "a waiting upload sees the exhausted slot only after persistence succeeds",
			firstStatus:  http.StatusOK,
			secondStatus: http.StatusUnauthorized,
		},
		{
			explanation:  "a persistence failure releases the available slot for the waiting upload",
			insertError:  errors.New("failed to persist upload"),
			firstStatus:  http.StatusInternalServerError,
			secondStatus: http.StatusOK,
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
				secondDone := false
				second := httptest.NewRecorder()
				{
					body, contentType := createMultipartFormBody("second.txt", "", "", strings.NewReader("second payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					go func() {
						s.Router().ServeHTTP(second, req)
						secondDone = true
					}()
				}
				synctest.Wait()
				if got, want := secondDone, false; got != want {
					t.Errorf("request finished before persistence=%v, want=%v", got, want)
				}
				releaseFirst()
				synctest.Wait()
				if got, want := first.Code, tt.firstStatus; got != want {
					t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
				}
				if got, want := second.Code, tt.secondStatus; got != want {
					t.Fatalf("second status=%d, want=%d: %s", got, want, second.Body.String())
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

func TestCanceledGuestUploadStopsWaiting(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(2),
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

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		canceledRead := false
		canceledDone := false
		{
			body, contentType := createMultipartFormBody("canceled.txt", "", "", strings.NewReader("canceled payload"))
			observedBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() { canceledRead = true }}
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/guest/abcdefgh23456789", observedBody)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			go func() {
				s.Router().ServeHTTP(rec, req)
				canceledDone = true
			}()
		}
		synctest.Wait()
		cancel()
		synctest.Wait()
		if got, want := canceledDone, true; got != want {
			t.Errorf("canceled request finished=%v, want=%v", got, want)
		}
		if got, want := canceledRead, false; got != want {
			t.Errorf("canceled request read its body=%v, want=%v", got, want)
		}
		releaseFirst()
		synctest.Wait()
		if got, want := first.Code, http.StatusOK; got != want {
			t.Fatalf("first status=%d, want=%d: %s", got, want, first.Body.String())
		}
		{
			body, contentType := createMultipartFormBody("third.txt", "", "", strings.NewReader("third payload"))
			req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("later upload status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		}
		entries, err := dataStore.GetEntriesMetadata()
		if err != nil {
			t.Fatalf("failed to list uploads: %v", err)
		}
		if got, want := len(entries), 2; got != want {
			t.Errorf("stored uploads=%d, want=%d", got, want)
		}
	})
}

func TestWaitingGuestUploadRechecksLink(t *testing.T) {
	for _, tt := range []struct {
		explanation  string
		changeLink   func(*testing.T, http.Handler)
		lookupError  error
		advanceTime  time.Duration
		secondStatus int
		laterStatus  int
	}{
		{
			explanation: "disabling a link admits the running upload but rejects the waiting upload",
			changeLink: func(t *testing.T, router http.Handler) {
				req := httptest.NewRequest(http.MethodPut, "/api/guest-links/abcdefgh23456789/disable", http.NoBody)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if got, want := rec.Code, http.StatusNoContent; got != want {
					t.Fatalf("disable status=%d, want=%d: %s", got, want, rec.Body.String())
				}
			},
			secondStatus: http.StatusUnauthorized,
			laterStatus:  http.StatusOK,
		},
		{
			explanation:  "expiration while waiting rejects the waiting upload but not the running upload",
			changeLink:   func(*testing.T, http.Handler) {},
			advanceTime:  2 * time.Hour,
			secondStatus: http.StatusUnauthorized,
			laterStatus:  http.StatusUnauthorized,
		},
		{
			explanation:  "a lookup failure after waiting releases the lock for later uploads",
			changeLink:   func(*testing.T, http.Handler) {},
			lookupError:  errors.New("guest link lookup failed"),
			secondStatus: http.StatusInternalServerError,
			laterStatus:  http.StatusOK,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			now := mustParseTime("2025-01-01T00:00:00Z")
			mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      mustParseExpirationTime("2025-01-01T01:00:00Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileUploads:  new(3),
			})
			synctest.Test(t, func(t *testing.T) {
				var lookupError error
				lookupStore := guestUploadLookupStore{Store: &dataStore, lookup: func(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
					if lookupError != nil {
						return picoshare.GuestLink{}, lookupError
					}
					return dataStore.GetGuestLink(id)
				}}
				s := handlers.New(mockAuthenticator{}, lookupStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
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
				secondRead := false
				second := httptest.NewRecorder()
				{
					body, contentType := createMultipartFormBody("second.txt", "", "", strings.NewReader("second payload"))
					observedBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() { secondRead = true }}
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", observedBody)
					req.Header.Set("Content-Type", contentType)
					go s.Router().ServeHTTP(second, req)
				}
				synctest.Wait()
				tt.changeLink(t, s.Router())
				now = now.Add(tt.advanceTime)
				lookupError = tt.lookupError
				releaseFirst()
				synctest.Wait()
				if got, want := first.Code, http.StatusOK; got != want {
					t.Fatalf("admitted upload status=%d, want=%d: %s", got, want, first.Body.String())
				}
				if got, want := second.Code, tt.secondStatus; got != want {
					t.Fatalf("waiting upload status=%d, want=%d: %s", got, want, second.Body.String())
				}
				if got, want := secondRead, false; got != want {
					t.Errorf("rejected request read its body=%v, want=%v", got, want)
				}
				entries, err := dataStore.GetEntriesMetadata()
				if err != nil {
					t.Fatalf("failed to list uploads: %v", err)
				}
				if got, want := len(entries), 1; got != want {
					t.Errorf("stored uploads=%d, want=%d", got, want)
				}
				lookupError = nil
				if err := dataStore.EnableGuestLink("abcdefgh23456789"); err != nil {
					t.Fatalf("failed to re-enable guest link: %v", err)
				}
				// A later request can acquire the lock again. An expired link
				// remains rejected; the other links can accept another upload.
				{
					body, contentType := createMultipartFormBody("third.txt", "", "", strings.NewReader("third payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, tt.laterStatus; got != want {
						t.Fatalf("later request status=%d, want=%d: %s", got, want, rec.Body.String())
					}
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
