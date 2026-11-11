package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/random"
	"github.com/mtlynch/picoshare/store/sqlite"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type guestLifecycleReader struct {
	reader     io.Reader
	beforeRead func()
}

func (r *guestLifecycleReader) Read(p []byte) (int, error) {
	if r.beforeRead != nil {
		callback := r.beforeRead
		r.beforeRead = nil
		callback()
	}
	return r.reader.Read(p)
}

type guestAbortFailureStore struct {
	handlers.Store
	reservation picoshare.GuestUploadReservation
	abortErr    error
}

func (s *guestAbortFailureStore) ReserveGuestUpload(ctx context.Context, id picoshare.GuestLinkID) (picoshare.GuestUploadReservation, error) {
	reservation, err := s.Store.ReserveGuestUpload(ctx, id)
	if err == nil {
		s.reservation = reservation
	}
	return reservation, err
}

func (s *guestAbortFailureStore) AbortGuestUpload(picoshare.GuestUploadReservation) error {
	return s.abortErr
}

func TestGuestUploadRevokedDuringRequest(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		revoke      func(*sqlite.Store, *time.Time, picoshare.GuestLinkID) error
		status      int
	}{
		{
			explanation: "disabling a reserved link should reject finalization and release its reservation",
			revoke: func(db *sqlite.Store, _ *time.Time, id picoshare.GuestLinkID) error {
				return db.DisableGuestLink(id)
			},
			status: http.StatusUnauthorized,
		},
		{
			explanation: "a link expiring during its upload should reject finalization and release its reservation",
			revoke: func(_ *sqlite.Store, now *time.Time, _ picoshare.GuestLinkID) error {
				*now = now.Add(time.Hour)
				return nil
			},
			status: http.StatusUnauthorized,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			db := sqlite.New(sqlite.Params{
				Path:               fmt.Sprintf("file:%s?mode=memory&cache=shared", random.String(10, []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))),
				Now:                func() time.Time { return now },
				PicoShareChunkSize: 2,
			})
			limit := 1
			link := picoshare.GuestLink{
				ID: "abcdefgh23456789", Created: now,
				UrlExpires:      picoshare.ExpirationTime(now.Add(time.Hour)),
				MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: &limit,
			}
			if err := db.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			s := handlers.New(mockAuthenticator{}, &db, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
			{
				body, contentType := createMultipartFormBody("revoked.txt", "", "", strings.NewReader("streamed before finalization"))
				reader := &guestLifecycleReader{reader: body, beforeRead: func() {
					reserved, err := db.GetGuestLink(link.ID)
					if err != nil {
						t.Fatal(err)
					}
					if got, want := reserved.UploadsReserved, 1; got != want {
						t.Fatalf("reservations before body read=%d, want=%d", got, want)
					}
					if err := tt.revoke(&db, &now, link.ID); err != nil {
						t.Fatal(err)
					}
				}}
				req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", reader)
				req.Header.Set("Content-Type", contentType)
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				if got, want := rec.Code, tt.status; got != want {
					t.Fatalf("status=%d, want=%d: %s", got, want, rec.Body.String())
				}
			}
			entries, err := db.GetEntriesMetadata()
			if err != nil {
				t.Fatal(err)
			}
			if got, want := len(entries), 0; got != want {
				t.Errorf("published entries=%d, want=%d", got, want)
			}
			updated, err := db.GetGuestLink(link.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := updated.UploadsReserved, 0; got != want {
				t.Errorf("remaining reservations=%d, want=%d", got, want)
			}
			if got, want := updated.FilesUploaded, 0; got != want {
				t.Errorf("completed uploads=%d, want=%d", got, want)
			}
		})
	}
}

func TestGuestUploadInterruptedBodyReleasesCapacity(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		interrupt   func(context.CancelFunc)
		readErr     error
		status      int
	}{
		{
			explanation: "a disconnected body should reject the upload and permit a retry",
			interrupt:   func(context.CancelFunc) {},
			readErr:     io.ErrUnexpectedEOF,
			status:      http.StatusBadRequest,
		},
		{
			explanation: "canceling a streaming request should clean up without its canceled context and permit a retry",
			interrupt:   func(cancel context.CancelFunc) { cancel() },
			readErr:     context.Canceled,
			status:      http.StatusBadRequest,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			db := test_sqlite.NewWithChunkSize(t, 256)
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			limit := 1
			link := picoshare.GuestLink{
				ID: "abcdefgh23456789", Created: now, UrlExpires: picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: &limit,
			}
			if err := db.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			s := handlers.New(mockAuthenticator{}, &db, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
			{
				body, contentType := createMultipartFormBody("interrupted.txt", "", "", strings.NewReader(strings.Repeat("partial upload", 1000)))
				payload, err := io.ReadAll(body)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				interruption := &guestLifecycleReader{
					reader: iotest.ErrReader(tt.readErr), beforeRead: func() { tt.interrupt(cancel) },
				}
				reader := io.MultiReader(bytes.NewReader(payload[:len(payload)/2]), interruption)
				req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", reader).WithContext(ctx)
				req.Header.Set("Content-Type", contentType)
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				if got, want := rec.Code, tt.status; got != want {
					t.Fatalf("status=%d, want=%d: %s", got, want, rec.Body.String())
				}
				entries, err := db.GetEntriesMetadata()
				if err != nil {
					t.Fatal(err)
				}
				if got, want := len(entries), 0; got != want {
					t.Errorf("published entries=%d, want=%d", got, want)
				}
			}
			{
				body, contentType := createMultipartFormBody("retry.txt", "", "", strings.NewReader("successful retry"))
				req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
				req.Header.Set("Content-Type", contentType)
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				if got, want := rec.Code, http.StatusOK; got != want {
					t.Fatalf("retry status=%d, want=%d: %s", got, want, rec.Body.String())
				}
			}
		})
	}
}

func TestGuestUploadAbortFailureRetainsCapacity(t *testing.T) {
	db := test_sqlite.NewWithChunkSize(t, 2)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	limit := 1
	link := picoshare.GuestLink{
		ID: "abcdefgh23456789", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: &limit,
	}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	dataStore := &guestAbortFailureStore{Store: &db, abortErr: errors.New("cleanup unavailable")}
	s := handlers.New(mockAuthenticator{}, dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	{
		body, contentType := createMultipartFormBody("failed.txt", "forbidden note after file", "", strings.NewReader("staged contents"))
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusInternalServerError; got != want {
			t.Fatalf("status=%d, want=%d: %s", got, want, rec.Body.String())
		}
		if got, want := strings.Contains(rec.Body.String(), "cleanup unavailable"), true; got != want {
			t.Errorf("response includes cleanup failure=%v, want=%v: %s", got, want, rec.Body.String())
		}
		updated, err := db.GetGuestLink(link.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := updated.UploadsReserved, 1; got != want {
			t.Errorf("retained reservations=%d, want=%d", got, want)
		}
		entries, err := db.GetEntriesMetadata()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := len(entries), 0; got != want {
			t.Errorf("published entries=%d, want=%d", got, want)
		}
	}
	{
		body := &observedUploadBody{}
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=unused")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("retry status=%d, want=%d: %s", got, want, rec.Body.String())
		}
		if got, want := body.read, false; got != want {
			t.Errorf("rejected retry body read=%v, want=%v", got, want)
		}
	}
	if err := db.AbortGuestUpload(dataStore.reservation); err != nil {
		t.Fatal(err)
	}
	{
		body, contentType := createMultipartFormBody("retry.txt", "", "", strings.NewReader("successful retry after cleanup"))
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("retry after cleanup status=%d, want=%d: %s", got, want, rec.Body.String())
		}
	}
}
