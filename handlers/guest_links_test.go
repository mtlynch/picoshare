package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

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
