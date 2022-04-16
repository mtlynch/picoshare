package handlers_test

import (
	"encoding/json"
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
	tests := []struct {
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
					"countLimit": null
				}`,
			expected: types.GuestLink{
				Label:                types.GuestLinkLabel(""),
				Expires:              mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes:         nil,
				UploadCountRemaining: nil,
			},
		},
		{
			description: "fully populated request",
			payload: `{
					"label": "For my good pal, Maurice",
					"expirationTime":"2030-01-02T03:04:25Z",
					"maxFileBytes": 200,
					"countLimit": 1
				}`,
			expected: types.GuestLink{
				Label:                types.GuestLinkLabel("For my good pal, Maurice"),
				Expires:              mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes:         makeGuestUploadMaxFileBytesPtr(200),
				UploadCountRemaining: makeGuestUploadCountLimitPtr(1),
			},
		},
	}
	for _, tt := range tests {
		dataStore := test_sqlite.New()

		s := handlers.New(mockAuthenticator{}, dataStore)

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
	}
}

func TestGuestLinksPostRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
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
			description: "invalid label field",
			payload: `{
					"label": 5,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"countLimit": null
				}`,
		},
		{
			description: "missing expirationTime field",
			payload: `{
					"label": null,
					"maxFileBytes": null,
					"countLimit": null
				}`,
		},
		{
			description: "invalid expirationTime field",
			payload: `{
					"label": null,
					"expirationTime": 25,
					"maxFileBytes": null,
					"countLimit": null
				}`,
		},
		{
			description: "invalid maxFileBytes field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": -5,
					"countLimit": null
				}`,
		},
		{
			description: "invalid countLimit field",
			payload: `{
					"label": null,
					"expirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"countLimit": -5
				}`,
		},
	}
	for _, tt := range tests {
		dataStore := test_sqlite.New()

		s := handlers.New(mockAuthenticator{}, dataStore)

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
	}
}

func makeGuestUploadMaxFileBytesPtr(i int64) *types.GuestUploadMaxFileBytes {
	g := types.GuestUploadMaxFileBytes(i)
	return &g
}

func makeGuestUploadCountLimitPtr(i int) *types.GuestUploadCountLimit {
	c := types.GuestUploadCountLimit(i)
	return &c
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	dataStore.InsertGuestLink(types.GuestLink{
		ID:      types.GuestLinkID("abcdefgh23456789"),
		Created: time.Now(),
		Expires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})

	s := handlers.New(mockAuthenticator{}, dataStore)

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	_, err = dataStore.GetGuestLink(types.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", types.EntryID("abcdefgh23456789"), err)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()

	s := handlers.New(mockAuthenticator{}, dataStore)

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
