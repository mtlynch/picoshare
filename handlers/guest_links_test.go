package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestGuestLinksPostAcceptsValidRequest(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
		expected    types.GuestLink
	}{
		{
			description: "minimally populated request",
			payload: `{
					"label": null,
					"expirationTime":"2030-01-02T03:04:25Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			expected: types.GuestLink{
				Label:          types.GuestLinkLabel(""),
				Expires:        mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes:   types.GuestUploadUnlimitedFileSize,
				MaxFileUploads: types.GuestUploadUnlimitedFileUploads,
			},
		},
		{
			description: "fully populated request",
			payload: `{
					"label": "For my good pal, Maurice",
					"expirationTime":"2030-01-02T03:04:25Z",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			expected: types.GuestLink{
				Label:          types.GuestLinkLabel("For my good pal, Maurice"),
				Expires:        mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes:   makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads: makeGuestUploadCountLimit(1),
			},
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()

			s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

			req, err := http.NewRequest("POST", "/api/guest-links", strings.NewReader(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", "text/json")

			w := httptest.NewRecorder()
			s.Router().ServeHTTP(w, req)

			if status := w.Code; status != http.StatusOK {
				t.Fatalf("%s: handler returned wrong status code: got %v want %v",
					tt.description, status, http.StatusOK)
			}

			var response handlers.GuestLinkPostResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("response is not valid JSON: %v", w.Body.String())
			}

			gl, err := dataStore.GetGuestLink(types.GuestLinkID(response.ID))
			if err != nil {
				t.Fatalf("%s: failed to retrieve guest link from datastore: %v", tt.description, err)
			}

			// Copy the values that we can't predict in advance.
			tt.expected.ID = types.GuestLinkID(response.ID)
			tt.expected.Created = gl.Created

			if !reflect.DeepEqual(gl, tt.expected) {
				t.Fatalf("%s: guest link does not match expected: got %+v, want %+v", tt.description, gl, tt.expected)
			}
		})
	}
}

func TestGuestLinksPostRejectsInvalidRequest(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
	}{
		{
			description: "empty string",
			payload:     "",
		},
		{
			description: "empty payload",
			payload:     "{}",
		},
		{
			description: "invalid label field (non-string)",
			payload: `{
					"label": 5,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
		},
		{
			description: "invalid label field (too long)",
			payload: fmt.Sprintf(`{
					"label": "%s",
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`, strings.Repeat("A", 201)),
		},
		{
			description: "missing expirationTime field",
			payload: `{
					"label": null,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
		},
		{
			description: "invalid expirationTime field",
			payload: `{
					"label": null,
					"expirationTime": 25,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
		},
		{
			description: "negative maxFileBytes field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": -5,
					"maxFileUploads": null
				}`,
		},
		{
			description: "decimal maxFileBytes field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1.5,
					"maxFileUploads": null
				}`,
		},
		{
			description: "too low a maxFileBytes field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1,
					"maxFileUploads": null
				}`,
		},
		{
			description: "zero maxFileBytes field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 0,
					"maxFileUploads": null
				}`,
		},
		{
			description: "negative maxFileUploads field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": -5
				}`,
		},
		{
			description: "decimal maxFileUploads field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 1.5
				}`,
		},
		{
			description: "zero maxFileUploads field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 0
				}`,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()

			s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

			req, err := http.NewRequest("POST", "/api/guest-links", strings.NewReader(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", "text/json")

			w := httptest.NewRecorder()
			s.Router().ServeHTTP(w, req)

			if status := w.Code; status != http.StatusBadRequest {
				t.Fatalf("%s: handler returned wrong status code: got %v want %v",
					tt.description, status, http.StatusBadRequest)
			}
		})
	}
}

func makeGuestUploadMaxFileBytes(i uint64) types.GuestUploadMaxFileBytes {
	return types.GuestUploadMaxFileBytes(&i)
}

func makeGuestUploadCountLimit(i int) types.GuestUploadCountLimit {
	return types.GuestUploadCountLimit(&i)
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	dataStore.InsertGuestLink(types.GuestLink{
		ID:      types.GuestLinkID("abcdefgh23456789"),
		Created: time.Now(),
		Expires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	_, err = dataStore.GetGuestLink(types.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", types.EntryID("abcdefgh23456789"), err)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

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

	s := handlers.New(mockAuthenticator{}, dataStore, nilGarbageCollector)

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
