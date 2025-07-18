package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestGuestUploadView(t *testing.T) {
	authenticator, err := shared_secret.New("dummypass")
	if err != nil {
		t.Fatalf("failed to create shared secret: %v", err)
	}

	for _, tt := range []struct {
		description               string
		guestLinkInStore          picoshare.GuestLink
		currentTime               time.Time
		expectedExpirationOptions []string
		expectedDefaultExpiration string
	}{
		{
			description: "guest link with infinite file lifetime shows all options",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expectedExpirationOptions: []string{
				"1 day",
				"7 days",
				"30 days",
				"1 year",
				"Never",
			},
			expectedDefaultExpiration: "Never",
		},
		{
			description: "guest link with 7 day limit shows only options up to 7 days",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInDays(7),
			},
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expectedExpirationOptions: []string{
				"1 day",
				"7 days",
			},
			expectedDefaultExpiration: "7 days",
		},
		{
			description: "guest link with 30 day limit shows options up to 30 days",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInDays(30),
			},
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expectedExpirationOptions: []string{
				"1 day",
				"7 days",
				"30 days",
			},
			expectedDefaultExpiration: "30 days",
		},
		{
			description: "guest link with 1 year limit shows options up to 1 year",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInYears(1),
			},
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expectedExpirationOptions: []string{
				"1 day",
				"7 days",
				"30 days",
				"1 year",
			},
			expectedDefaultExpiration: "1 year",
		},
		{
			description: "guest link with 1 day limit shows only 1 day option",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInDays(1),
			},
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expectedExpirationOptions: []string{
				"1 day",
			},
			expectedDefaultExpiration: "1 day",
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			if err := dataStore.InsertGuestLink(tt.guestLinkInStore); err != nil {
				t.Fatalf("failed to insert dummy guest link: %v", err)
			}

			c := mockClock{tt.currentTime}
			s := handlers.New(authenticator, &dataStore, nilSpaceChecker, nilGarbageCollector, c)

			req, err := http.NewRequest("GET", "/g/"+tt.guestLinkInStore.ID.String(), nil)
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, http.StatusOK; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			// Read the response body to check the HTML content.
			responseBody := rec.Body.String()

			// Check that all expected expiration options are present.
			for _, option := range tt.expectedExpirationOptions {
				if !strings.Contains(responseBody, option) {
					t.Errorf("expected expiration option %q not found in response", option)
				}
			}

			// Check that options beyond the guest link's limit are not present.
			allOptions := []string{"1 day", "7 days", "30 days", "1 year", "Never"}
			for _, option := range allOptions {
				found := false
				for _, expected := range tt.expectedExpirationOptions {
					if option == expected {
						found = true
						break
					}
				}
				if !found && strings.Contains(responseBody, ">"+option+"<") {
					t.Errorf("expiration option %q should not be present but was found in response", option)
				}
			}

			// Check that the expected default is marked as selected.
			// Look for the option element containing both "selected" and the expected text.
			// The HTML format has the selected attribute on a separate line.
			lines := strings.Split(responseBody, "\n")
			found := false
			for i, line := range lines {
				if strings.Contains(line, ">"+tt.expectedDefaultExpiration+"<") {
					// Found the line with the option text, check nearby lines for "selected" attribute.
					for j := i - 3; j <= i; j++ {
						if j >= 0 && j < len(lines) && strings.Contains(lines[j], "selected") {
							found = true
							break
						}
					}
					break
				}
			}
			if !found {
				t.Errorf("expected default expiration option %q not marked as selected in response", tt.expectedDefaultExpiration)
			}

			// Check that the expiration dropdown container is present.
			if !strings.Contains(responseBody, `class="field my-5 expiration-container"`) {
				t.Error("expiration container not found in guest upload view")
			}

			// Check that the expiration select element is present.
			if !strings.Contains(responseBody, `id="expiration-select"`) {
				t.Error("expiration select element not found in guest upload view")
			}
		})
	}
}
