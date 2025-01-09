package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
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
				Created:        mustParseTime("2024-01-01T00:00:00Z"),
				Label:          picoshare.GuestLinkLabel(""),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
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
				Created:        mustParseTime("2024-01-01T00:00:00Z"),
				Label:          picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads: makeGuestUploadCountLimit(1),
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
				Created:        mustParseTime("2024-01-01T00:00:00Z"),
				Label:          picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.NewFileLifetimeInDays(1),
				MaxFileBytes:   makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads: makeGuestUploadCountLimit(1),
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
				Created:        mustParseTime("2024-01-01T00:00:00Z"),
				Label:          picoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.NewFileLifetimeInDays(30),
				MaxFileBytes:   makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads: makeGuestUploadCountLimit(1),
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
			dataStore := test_sqlite.New()
			c := mockClock{tt.currentTime}
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, c)

			req, err := http.NewRequest("POST", "/api/guest-links", strings.NewReader(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
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

func makeGuestUploadMaxFileBytes(i uint64) picoshare.GuestUploadMaxFileBytes {
	return picoshare.GuestUploadMaxFileBytes(&i)
}

func makeGuestUploadCountLimit(i int) picoshare.GuestUploadCountLimit {
	return picoshare.GuestUploadCountLimit(&i)
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	dataStore.InsertGuestLink(picoshare.GuestLink{
		ID:         picoshare.GuestLinkID("abcdefgh23456789"),
		Created:    time.Now(),
		UrlExpires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}

	_, err = dataStore.GetGuestLink(picoshare.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", picoshare.EntryID("abcdefgh23456789"), err)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/guest-links/abcdefgh23456789", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestDeleteInvalidGuestLink(t *testing.T) {
	dataStore := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	req, err := http.NewRequest("DELETE", "/api/guest-links/i-am-an-invalid-link", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestGuestLinkToggleInValidRequest(t *testing.T) {

	type operation struct {
		name   string
		action func(s handlers.Server, id string) (*httptest.ResponseRecorder, error)
	}

	type testCase struct {
		description string
		payload     string
		operations  []operation
		expected    picoshare.GuestLink
	}

	// Create an operation that can be either 'enable' or 'disable' a guest link.
	createOperation := func(action string) operation {
		return operation{
			name: action,
			action: func(s handlers.Server, id string) (*httptest.ResponseRecorder, error) {
				req, err := http.NewRequest("PUT", fmt.Sprintf("/api/guest-links/%s/%s", id, action), nil)
				if err != nil {
					return nil, err
				}
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				return rec, nil
			},
		}
	}

	disableOperation := createOperation("disable")
	enableOperation := createOperation("enable")

	//Function to create a new guest link.
	createLink := func(s handlers.Server, payload string) (*handlers.GuestLinkPostResponse, *httptest.ResponseRecorder, error) {
		req, err := http.NewRequest("POST", "/api/guest-links", strings.NewReader(payload))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Add("Content-Type", "text/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		res := rec.Result()

		if got, want := res.StatusCode, http.StatusOK; got != want {
			return nil, rec, fmt.Errorf("unexpected status code: %d", got)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, rec, err
		}

		var response handlers.GuestLinkPostResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, rec, err
		}

		return &response, rec, nil
	}

	for _, tt := range []testCase{
		{
			description: "guest_link operation: disable",
			payload: `{
					"label": "disable",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			operations: []operation{disableOperation},
			expected: picoshare.GuestLink{
				Label:          picoshare.GuestLinkLabel("disable"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:     true,
			},
		},
		{
			description: "guest_link operation: enable",
			payload: `{
					"label": "enable",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			operations: []operation{enableOperation},
			expected: picoshare.GuestLink{
				Label:          picoshare.GuestLinkLabel("enable"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:     false,
			},
		},
		{
			description: "guest_link operations: disable and enable",
			payload: `{
					"label": "disable and enable",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			operations: []operation{disableOperation, enableOperation},
			expected: picoshare.GuestLink{
				Label:          picoshare.GuestLinkLabel("disable and enable"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:     false,
			},
		},
		{
			description: "guest_link operations: enable and disable",
			payload: `{
					"label": "enable and disable",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			operations: []operation{enableOperation, disableOperation},
			expected: picoshare.GuestLink{
				Label:          picoshare.GuestLinkLabel("enable and disable"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime:   picoshare.FileLifetimeInfinite,
				MaxFileBytes:   picoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:     true,
			},
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

			// Create the guest link based on the provided payload
			response, rec, err := createLink(s, tt.payload)
			if err != nil {
				t.Fatalf("failed to create guest link: %v", err)
			}
			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("failed to create guest link: %v", rec.Code)
			}

			// Execute the operations defined in the test case.
			for _, op := range tt.operations {
				rec, err := op.action(s, response.ID)
				if err != nil {
					t.Fatalf("failed to %s guest link: %v", op.name, err)
				}
				if got, want := rec.Code, http.StatusNoContent; got != want {
					t.Fatalf("failed to %s guest link: %v", op.name, rec.Code)
				}
			}

			gl, err := dataStore.GetGuestLink(picoshare.GuestLinkID(response.ID))
			if err != nil {
				t.Fatalf("failed to retrieve guest link from datastore: %v", err)
			}

			// Copy the values that we can't predict in advance.
			tt.expected.Created = gl.Created
			tt.expected.ID = picoshare.GuestLinkID(response.ID)

			if !reflect.DeepEqual(gl, tt.expected) {
				t.Errorf("guest link does not match expected: got %+v, want %+v", gl, tt.expected)
			}
		})
	}
}

func TestGuestLinkOperations(t *testing.T) {
	type operation struct {
		name   string
		action func(s handlers.Server, id string) (*httptest.ResponseRecorder, error)
	}

	// Create an operation that can be either 'enable' or 'disable' a guest link.
	createOperation := func(action string) operation {
		return operation{
			name: action,
			action: func(s handlers.Server, id string) (*httptest.ResponseRecorder, error) {
				req, err := http.NewRequest("PUT", fmt.Sprintf("/api/guest-links/%s/%s", id, action), nil)
				if err != nil {
					return nil, err
				}
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				return rec, nil
			},
		}
	}

	disableOperation := createOperation("disable")
	enableOperation := createOperation("enable")
	for _, tt := range []struct {
		description string
		payload     picoshare.GuestLinkID
		operation   operation
		expected    int
	}{
		{
			description: "disable a nonexistent guest link",
			payload:     "abcdefgh23456789",
			operation:   disableOperation,
			expected:    http.StatusNotFound,
		},
		{
			description: "enable a nonexistent guest link",
			payload:     "abcdefgh23456789",
			operation:   enableOperation,
			expected:    http.StatusNotFound,
		},
		{
			description: "disable an invalid guest link",
			payload:     "i-am-an-invalid-link",
			operation:   disableOperation,
			expected:    http.StatusNotFound,
		},
		{
			description: "enable an invalid guest link",
			payload:     "i-am-an-invalid-link",
			operation:   enableOperation,
			expected:    http.StatusNotFound,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())
			// Execute the operation defined in the test case.
			rec, err := tt.operation.action(s, string(tt.payload))
			if err != nil {
				t.Fatalf("failed to %s guest link: %v", tt.operation.name, err)
			}
			if got, want := rec.Code, tt.expected; got != want {
				t.Errorf("failed to %s guest link: got %d, want %d", tt.operation.name, got, want)
			}
		})
	}
}
