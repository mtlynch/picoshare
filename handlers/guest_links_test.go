package handlers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestGuestLinksPost(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
		currentTime time.Time
		expected    picoshare.GuestLink
		status      int
	}{
		{
			description: "minimally populated request",
			payload: `{
					"label": null,
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: picoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           picoshare.GuestLinkLabel(""),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
			},
			status: http.StatusOK,
		},
		{
			description: "fully populated request",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: picoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "guest file expires in 1 day",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"24h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: picoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(1),
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "guest file expires in 30 days",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"720h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: picoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(30),
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "empty string",
			payload:     "",
			status:      http.StatusBadRequest,
		},
		{
			description: "empty payload",
			payload:     "{}",
			status:      http.StatusBadRequest,
		},
		{
			description: "invalid label field (non-string)",
			payload: `{
					"label": 5,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "invalid label field (too long)",
			payload: fmt.Sprintf(`{
					"label": "%s",
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`, strings.Repeat("A", 201)),
			status: http.StatusBadRequest,
		},
		{
			description: "missing urlExpirationTime field",
			payload: `{
					"label": null,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "invalid expirationTime field",
			payload: `{
					"label": null,
					"urlExpirationTime": 25,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "negative maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": -5,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "decimal maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1.5,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "too low a maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "zero maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 0,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "negative maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": -5
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "decimal maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 1.5
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "zero maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 0
				}`,
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			now := tt.currentTime
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/guest-links",
				strings.NewReader(tt.payload),
			)
			req.Header.Add("Content-Type", "text/json")

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			if tt.status != http.StatusOK {
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed to read response body")
			}

			var response handlers.GuestLinkPostResponse
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatalf("response is not valid JSON: %v", body)
			}

			gl, err := dataStore.GetGuestLink(picoshare.GuestLinkID(response.ID))
			if err != nil {
				t.Fatalf("failed to retrieve guest link from datastore: %v", err)
			}

			// Copy the ID, which we can't predict in advance.
			tt.expected.ID = picoshare.GuestLinkID(response.ID)

			if got, want := gl, tt.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("guestLink=%+v, want=%+v", got, want)
			}
		})
	}
}

type failingUploadStore struct {
	handlers.Store
	insertEntryError error
}

func (s failingUploadStore) InsertEntry(io.Reader, picoshare.UploadMetadata) error {
	return s.insertEntryError
}

func TestGuestUploadPostPersistenceError(t *testing.T) {
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
		insertEntryError: errors.New("dummy database failure"),
	}
	s := handlers.New(mockAuthenticator{}, failingStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	body, contentType := createMultipartFormBody("upload.txt", "", "", strings.NewReader("payload"))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/guest/%s", guestLinkID), body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.Router().ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
	}
	if got, want := strings.TrimSpace(rec.Body.String()), "failed to insert file into database"; got != want {
		t.Errorf("response=%q, want=%q", got, want)
	}
}

func TestOwnerUploadPostPersistenceError(t *testing.T) {
	now := mustParseTime("2025-01-01T00:00:00Z")
	dataStore := test_sqlite.New(t)
	failingStore := failingUploadStore{
		Store:            &dataStore,
		insertEntryError: errors.New("dummy database failure"),
	}
	s := handlers.New(mockAuthenticator{}, failingStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	body, contentType := createMultipartFormBody("upload.txt", "", "", strings.NewReader("payload"))
	req := httptest.NewRequest(http.MethodPost, "/api/entry?expiration=2030-01-01T00:00:00Z", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	s.Router().ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
	}
	if got, want := strings.TrimSpace(rec.Body.String()), "failed to insert file into database"; got != want {
		t.Errorf("response=%q, want=%q", got, want)
	}
}

func makeGuestUploadMaxFileBytes(i uint64) picoshare.GuestUploadMaxFileBytes {
	return picoshare.GuestUploadMaxFileBytes(&i)
}

func makeGuestUploadCountLimit(i int) picoshare.GuestUploadCountLimit {
	return picoshare.GuestUploadCountLimit(&i)
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	dataStore.InsertGuestLink(picoshare.GuestLink{
		ID:         picoshare.GuestLinkID("abcdefgh23456789"),
		Created:    mustParseTime("2025-05-25T00:00:00Z"),
		UrlExpires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/abcdefgh23456789",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}

	_, err := dataStore.GetGuestLink(picoshare.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", "abcdefgh23456789", err)
	}
}

func TestDeletingGuestLinkRejectsInFlightUpload(t *testing.T) {
	// Create an active link so the upload passes its initial authorization check.
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

	// Withhold the body until the handler first tries to read it. Delete the
	// link at that point, then release the body to resume the in-flight upload.
	body, contentType := createMultipartFormBody("held.txt", "", "", strings.NewReader("held payload"))
	heldBody := &beforeReadReader{
		Reader: body,
		beforeRead: func() {
			req := httptest.NewRequest(http.MethodDelete, "/api/guest-links/abcdefgh23456789", http.NoBody)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("delete status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		},
	}

	// Start the upload while the link is active. Even though it was initially
	// authorized, it must be rejected after the link is deleted during the read.
	{
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", heldBody)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
		}
	}

	// Rejection must also prevent the uploaded file from being persisted.
	entries, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if got, want := len(entries), 0; got != want {
		t.Errorf("stored entries=%d, want=%d", got, want)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/abcdefgh23456789",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestDeleteInvalidGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/i-am-an-invalid-link",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestDisablingGuestLinkRejectsInFlightUpload(t *testing.T) {
	// Create an active link so the upload passes its initial authorization check.
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

	// Withhold the body until the handler first tries to read it. Disable the
	// link at that point, then release the body to resume the in-flight upload.
	body, contentType := createMultipartFormBody("held.txt", "", "", strings.NewReader("held payload"))
	heldBody := &beforeReadReader{
		Reader: body,
		beforeRead: func() {
			req := httptest.NewRequest(http.MethodPut, "/api/guest-links/abcdefgh23456789/disable", http.NoBody)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if got, want := rec.Code, http.StatusNoContent; got != want {
				t.Fatalf("disable status=%d, want=%d: %s", got, want, rec.Body.String())
			}
		},
	}

	// Start the upload while the link is active. Even though it was initially
	// authorized, it must be rejected after the link is disabled during the read.
	{
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", heldBody)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("upload status=%d, want=%d: %s", got, want, rec.Body.String())
		}
	}

	// Rejection must also prevent the uploaded file from being persisted.
	entries, err := dataStore.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if got, want := len(entries), 0; got != want {
		t.Errorf("stored entries=%d, want=%d", got, want)
	}
}

func TestEnableDisableGuestLink(t *testing.T) {
	for _, tt := range []struct {
		description      string
		requestRoute     string
		guestLinkInStore picoshare.GuestLink
		expected         picoshare.GuestLink
		status           int
	}{
		{
			description:  "disabling active guest link succeeds",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			expected: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "enabling inactive guest link succeeds",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			expected: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "disabling guest link succeeds but has no effect when guest link is not disabled",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			expected: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "enabling guest link succeeds but has no effect when guest link is not enabled",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			expected: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "disable a non-existent guest link",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			status:       http.StatusNotFound,
		}, {
			description:  "enable a non-existent guest link",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			status:       http.StatusNotFound,
		},
		{
			description:  "disable a guest link with an invalid ID",
			requestRoute: "/api/guest-links/i-am-an-invalid-link/disable",
			status:       http.StatusBadRequest,
		},
		{
			description:  "enable a guest link with an invalid ID",
			requestRoute: "/api/guest-links/i-am-an-invalid-link/enable",
			status:       http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)

			if !tt.guestLinkInStore.Empty() {
				if err := dataStore.InsertGuestLink(tt.guestLinkInStore); err != nil {
					t.Fatalf("failed to insert dummy guest link: %v", err)
				}
			}

			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			req := httptest.NewRequest(http.MethodPut, tt.requestRoute, nil)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			if tt.status != http.StatusNoContent {
				return
			}

			gl, err := dataStore.GetGuestLink(tt.guestLinkInStore.ID)
			if err != nil {
				t.Fatalf("failed to getGuestLink : %s", tt.guestLinkInStore.ID)
			}

			if got, want := gl, tt.expected; !reflect.DeepEqual(got, want) {
				t.Errorf("guestLink=%+v, want=%+v", got, want)
			}
		})
	}
}
