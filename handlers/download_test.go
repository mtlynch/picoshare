package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type mockEntry struct {
	ID          picoshare.EntryID
	Filename    picoshare.Filename
	ContentType picoshare.ContentType
}

var (
	dummyTextEntry = mockEntry{
		ID:          "TTTTTTTTTT",
		Filename:    picoshare.Filename("test.txt"),
		ContentType: picoshare.ContentType("text/plain;charset=utf-8"),
	}
	dummyAudioEntry = mockEntry{
		ID:          "AAAAAAAAAA",
		Filename:    picoshare.Filename("test.mp3"),
		ContentType: picoshare.ContentType("audio/mpeg"),
	}
	dummyAudioEntryWithoutContentType = mockEntry{
		ID:       "AAAAAAAA22",
		Filename: picoshare.Filename("test0.mp3"),
	}
	dummyVideoEntry = mockEntry{
		ID:          "VVVVVVVVVV",
		Filename:    picoshare.Filename("test.mp4"),
		ContentType: picoshare.ContentType("video/mp4"),
	}
	dummyVideoEntryWithGenericContentType = mockEntry{
		ID:          "VVVVVVVV22",
		Filename:    picoshare.Filename("test0.mp4"),
		ContentType: picoshare.ContentType("application/octet-stream"),
	}
	dummyHTMLEntry = mockEntry{
		ID:          "HHHHHHHHHH",
		Filename:    picoshare.Filename("payload.html"),
		ContentType: picoshare.ContentType("text/html"),
	}
)

func TestEntryGet(t *testing.T) {
	for _, tt := range []struct {
		description                string
		requestRoute               string
		expectedStatus             int
		expectedContentDisposition string
		expectedContentType        string
		expectedCSP                string
	}{
		{
			description:                "retrieves text entry",
			requestRoute:               "/-TTTTTTTTTT",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.txt"`,
			expectedContentType:        "text/plain;charset=utf-8",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves audio entry",
			requestRoute:               "/-AAAAAAAAAA",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.mp3"`,
			expectedContentType:        "audio/mpeg",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves audio entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-AAAAAAAA22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp3"`,
			expectedContentType:        "audio/mpeg",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves video entry",
			requestRoute:               "/-VVVVVVVVVV",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.mp4"`,
			expectedContentType:        "video/mp4",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves video entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-VVVVVVVV22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp4"`,
			expectedContentType:        "video/mp4",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves html entry",
			requestRoute:               "/-HHHHHHHHHH",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="payload.html"`,
			expectedContentType:        "text/html",
			expectedCSP:                "sandbox",
		},
		{
			description:    "request for non-existent entry returns 404",
			requestRoute:   "/-ZZZZZZZZZZ",
			expectedStatus: http.StatusNotFound,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)

			for _, mockEntry := range []mockEntry{
				dummyTextEntry,
				dummyAudioEntry,
				dummyAudioEntryWithoutContentType,
				dummyVideoEntry,
				dummyVideoEntryWithGenericContentType,
				dummyHTMLEntry,
			} {
				data := "dummy data"
				entry := picoshare.UploadEntry{
					UploadMetadata: picoshare.UploadMetadata{
						ID:          mockEntry.ID,
						Filename:    mockEntry.Filename,
						ContentType: mockEntry.ContentType,
						Uploaded:    mustParseTime("2023-01-01T00:00:00Z"),
						Expires:     picoshare.NeverExpire,
						Size:        mustParseFileSize(len(data)),
					},
					Reader: strings.NewReader(data),
				}
				if err := dataStore.InsertEntry(entry.Reader, entry.UploadMetadata); err != nil {
					panic(err)
				}
			}

			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

			req := httptest.NewRequest(http.MethodGet, tt.requestRoute, nil)

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.expectedStatus; got != want {
				t.Fatalf("%s returned wrong status code: got %v want %v",
					tt.requestRoute, got, want)
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			if got, want := res.Header.Get("Content-Disposition"), tt.expectedContentDisposition; got != want {
				t.Errorf("Content-Disposition=%s, want=%s", got, want)
			}

			if got, want := res.Header.Get("Content-Type"), tt.expectedContentType; got != want {
				t.Errorf("Content-Type=%s, want=%s", got, want)
			}

			if got, want := res.Header.Get("Content-Security-Policy"), tt.expectedCSP; got != want {
				t.Errorf("Content-Security-Policy=%s, want=%s", got, want)
			}
		})
	}
}

func TestProtectedEntryDownload(t *testing.T) {
	protectedData := "protected file contents"
	passphrase := mustCreatePassphrase(t, "correct horse battery staple")
	for _, tt := range []struct {
		explanation      string
		authenticated    bool
		method           string
		route            string
		passphrase       string
		expectedStatus   int
		expectedLocation string
		expectedCSP      string
		expectedBody     string
	}{
		{
			explanation:      "unauthenticated GET of a protected entry redirects to the unlock page",
			authenticated:    false,
			method:           http.MethodGet,
			route:            "/-PPPPPPPPPP",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-PPPPPPPPPP/unlock",
		},
		{
			explanation:      "unauthenticated GET of a protected entry with a filename redirects to the unlock page",
			authenticated:    false,
			method:           http.MethodGet,
			route:            "/-PPPPPPPPPP/protected.txt",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-PPPPPPPPPP/unlock",
		},
		{
			explanation:      "unauthenticated GET of a protected entry via a legacy route redirects to the unlock page",
			authenticated:    false,
			method:           http.MethodGet,
			route:            "/!PPPPPPPPPP",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-PPPPPPPPPP/unlock",
		},
		{
			explanation:    "unauthenticated GET of the unlock page renders the challenge with a nonce CSP",
			authenticated:  false,
			method:         http.MethodGet,
			route:          "/-PPPPPPPPPP/unlock",
			expectedStatus: http.StatusOK,
			expectedCSP:    "nonce",
			expectedBody:   "Protected Download",
		},
		{
			explanation:    "incorrect passphrase re-renders the challenge with 401",
			authenticated:  false,
			method:         http.MethodPost,
			route:          "/-PPPPPPPPPP/unlock",
			passphrase:     "wrong passphrase",
			expectedStatus: http.StatusUnauthorized,
			expectedCSP:    "nonce",
			expectedBody:   "Incorrect passphrase.",
		},
		{
			explanation:    "correct passphrase serves the file with sandbox CSP",
			authenticated:  false,
			method:         http.MethodPost,
			route:          "/-PPPPPPPPPP/unlock",
			passphrase:     "correct horse battery staple",
			expectedStatus: http.StatusOK,
			expectedCSP:    "sandbox",
			expectedBody:   protectedData,
		},
		{
			explanation:    "POST to the download route is not allowed",
			authenticated:  false,
			method:         http.MethodPost,
			route:          "/-PPPPPPPPPP",
			passphrase:     "correct horse battery staple",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			explanation:    "authenticated owner downloads a protected entry without a challenge",
			authenticated:  true,
			method:         http.MethodGet,
			route:          "/-PPPPPPPPPP",
			expectedStatus: http.StatusOK,
			expectedCSP:    "sandbox",
			expectedBody:   protectedData,
		},
		{
			explanation:      "authenticated owner visiting the unlock page redirects to the download",
			authenticated:    true,
			method:           http.MethodGet,
			route:            "/-PPPPPPPPPP/unlock",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-PPPPPPPPPP",
		},
		{
			explanation:      "unlock page for an unprotected entry redirects to the download",
			authenticated:    false,
			method:           http.MethodGet,
			route:            "/-TTTTTTTTTT/unlock",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-TTTTTTTTTT",
		},
		{
			explanation:    "unlock page for a non-existent entry returns 404",
			authenticated:  false,
			method:         http.MethodGet,
			route:          "/-ZZZZZZZZZZ/unlock",
			expectedStatus: http.StatusNotFound,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			if err := dataStore.InsertEntry(strings.NewReader(protectedData), picoshare.UploadMetadata{
				ID:                 "PPPPPPPPPP",
				Filename:           "protected.txt",
				ContentType:        "text/plain",
				Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
				Expires:            picoshare.NeverExpire,
				Size:               mustParseFileSize(len(protectedData)),
				DownloadPassphrase: &passphrase,
			}); err != nil {
				t.Fatalf("failed to insert protected entry: %v", err)
			}
			unprotectedData := "dummy data"
			if err := dataStore.InsertEntry(strings.NewReader(unprotectedData), picoshare.UploadMetadata{
				ID:          dummyTextEntry.ID,
				Filename:    dummyTextEntry.Filename,
				ContentType: dummyTextEntry.ContentType,
				Uploaded:    mustParseTime("2023-01-01T00:00:00Z"),
				Expires:     picoshare.NeverExpire,
				Size:        mustParseFileSize(len(unprotectedData)),
			}); err != nil {
				t.Fatalf("failed to insert unprotected entry: %v", err)
			}

			var authenticator handlers.Authenticator = unauthenticatedAuthenticator{}
			if tt.authenticated {
				authenticator = mockAuthenticator{}
			}
			s := handlers.New(authenticator, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

			var req *http.Request
			if tt.method == http.MethodPost {
				form := url.Values{"passphrase": {tt.passphrase}}
				req = httptest.NewRequest(tt.method, tt.route, strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(tt.method, tt.route, nil)
			}
			rec := httptest.NewRecorder()

			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.expectedStatus; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}
			if got := res.Header.Get("Set-Cookie"); got != "" {
				t.Errorf("Set-Cookie=%q, want empty", got)
			}
			if tt.expectedStatus == http.StatusNotFound || tt.expectedStatus == http.StatusMethodNotAllowed {
				return
			}
			if got, want := res.Header.Get("Cache-Control"), "no-store"; got != want {
				t.Errorf("Cache-Control=%q, want=%q", got, want)
			}
			if got, want := res.Header.Get("Location"), tt.expectedLocation; got != want {
				t.Errorf("Location=%q, want=%q", got, want)
			}
			switch tt.expectedCSP {
			case "sandbox":
				if got, want := res.Header.Get("Content-Security-Policy"), "sandbox"; got != want {
					t.Errorf("Content-Security-Policy=%q, want=%q", got, want)
				}
			case "nonce":
				if got := res.Header.Get("Content-Security-Policy"); got == "sandbox" || !strings.Contains(got, "'nonce-") {
					t.Errorf("Content-Security-Policy=%q, want nonce policy", got)
				}
			}
			if got := rec.Body.String(); !strings.Contains(got, tt.expectedBody) {
				t.Errorf("body=%q, want to contain %q", got, tt.expectedBody)
			}
		})
	}
}

func TestProtectedEntryDownloadDoesNotPersistUnlock(t *testing.T) {
	dataStore := test_sqlite.New(t)
	data := "protected file contents"
	passphrase := mustCreatePassphrase(t, "correct horse battery staple")
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:                 "PPPPPPPPPP",
		Filename:           "protected.txt",
		ContentType:        "text/plain",
		Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
		Expires:            picoshare.NeverExpire,
		Size:               mustParseFileSize(len(data)),
		DownloadPassphrase: &passphrase,
	}); err != nil {
		t.Fatalf("failed to insert protected entry: %v", err)
	}
	s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

	form := url.Values{"passphrase": {"correct horse battery staple"}}
	req := httptest.NewRequest(http.MethodPost, "/-PPPPPPPPPP/unlock", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
	if got, want := rec.Body.String(), data; got != want {
		t.Fatalf("body=%q, want=%q", got, want)
	}

	followUpReq := httptest.NewRequest(http.MethodGet, "/-PPPPPPPPPP", nil)
	followUpRec := httptest.NewRecorder()
	s.Router().ServeHTTP(followUpRec, followUpReq)
	if got, want := followUpRec.Code, http.StatusFound; got != want {
		t.Errorf("follow-up status=%d, want=%d", got, want)
	}
	if got, want := followUpRec.Header().Get("Location"), "/-PPPPPPPPPP/unlock"; got != want {
		t.Errorf("follow-up Location=%q, want=%q", got, want)
	}
}

func mustCreatePassphrase(t *testing.T, value string) picoshare.Passphrase {
	t.Helper()

	passphrase, err := picoshare.NewPassphrase(value)
	if err != nil {
		t.Fatalf("failed to create passphrase: %v", err)
	}

	return passphrase
}
