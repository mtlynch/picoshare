package handlers_test

import (
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
	"github.com/mtlynch/picoshare/testutil"
)

func TestAdmittedGuestUploadFinishesAfterLinkBecomesInactive(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		deactivate  func(*testing.T, http.Handler, *time.Time)
	}{
		{
			explanation: "disabling a link allows its admitted upload to finish but rejects subsequent uploads",
			deactivate: func(t *testing.T, router http.Handler, _ *time.Time) {
				req := httptest.NewRequest(http.MethodPut, "/api/guest-links/abcdefgh23456789/disable", http.NoBody)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if got, want := rec.Code, http.StatusNoContent; got != want {
					t.Fatalf("disable status=%d, want=%d: %s", got, want, rec.Body.String())
				}
			},
		},
		{
			explanation: "expiration allows an admitted upload to finish but rejects subsequent uploads",
			deactivate: func(_ *testing.T, _ http.Handler, now *time.Time) {
				*now = mustParseTime("2025-01-01T02:00:00Z")
			},
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			now := mustParseTime("2025-01-01T00:00:00Z")
			mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
				ID:              "abcdefgh23456789",
				Created:         now,
				UrlExpires:      testutil.MustParseExpirationTime(t, "2025-01-01T01:00:00Z"),
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
				synctest.Wait()
				tt.deactivate(t, s.Router(), &now)
				releaseFirst()
				synctest.Wait()
				if got, want := first.Code, http.StatusOK; got != want {
					t.Fatalf("admitted upload status=%d, want=%d: %s", got, want, first.Body.String())
				}
				{
					body, contentType := createMultipartFormBody("later.txt", "", "", strings.NewReader("later payload"))
					req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
					req.Header.Set("Content-Type", contentType)
					rec := httptest.NewRecorder()
					s.Router().ServeHTTP(rec, req)
					if got, want := rec.Code, http.StatusUnauthorized; got != want {
						t.Fatalf("later upload status=%d, want=%d: %s", got, want, rec.Body.String())
					}
					if got, want := rec.Body.String(), "Guest link is no longer active\n"; got != want {
						t.Errorf("rejection=%q, want=%q", got, want)
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
