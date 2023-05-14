package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestSettingsPut(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
		settings    picoshare.Settings
		status      int
	}{
		{
			description: "valid request",
			payload: `{
					"defaultExpirationDays": 7
				}`,
			settings: picoshare.Settings{
				DefaultFileLifetime: picoshare.NewFileLifetimeInDays(7),
			},
			status: http.StatusOK,
		},
		{
			description: "rejects invalid expiration days (too low)",
			payload: `{
					"defaultExpirationDays": 0
				}`,
			settings: picoshare.Settings{},
			status:   http.StatusBadRequest,
		},
		{
			description: "rejects invalid expiration days (not a number)",
			payload: `{
					"defaultExpirationDays": "banana"
				}`,
			settings: picoshare.Settings{},
			status:   http.StatusBadRequest,
		},
		{
			description: "rejects request with missing expiration days",
			payload: `{
					"someIrrelevantField": "banana"
				}`,
			settings: picoshare.Settings{},
			status:   http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			s := handlers.New(mockAuthenticator{}, dataStore, nilSpaceChecker, nilGarbageCollector)

			req, err := http.NewRequest("PUT", "/api/settings", strings.NewReader(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", "text/json")

			w := httptest.NewRecorder()
			s.Router().ServeHTTP(w, req)

			if got, want := w.Code, tt.status; got != want {
				t.Fatalf("/api/settings returned wrong status code: got %v want %v",
					got, want)
			}

			if tt.status != http.StatusOK {
				return
			}

			settings, err := dataStore.ReadSettings()
			if err != nil {
				t.Fatalf("failed to retrieve settings from datastore: %v", err)
			}

			if got, want := settings, tt.settings; !reflect.DeepEqual(got, want) {
				t.Fatalf("settings in datastore don't match payload: got %+v, want %+v", got, want)
			}
		})
	}
}
