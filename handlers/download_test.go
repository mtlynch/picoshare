package handlers_test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

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
	type fakeEntry struct {
		ID                 picoshare.EntryID
		Contents           string
		DownloadPassphrase picoshare.DownloadPassphrase
	}

	protectedEntry := fakeEntry{
		ID:                 "PPPPPPPPPP",
		Contents:           "fake protected data",
		DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
	}
	unprotectedEntry := fakeEntry{
		ID:       "UUUUUUUUUU",
		Contents: "fake unprotected data",
	}

	for _, tt := range []struct {
		explanation                string
		authenticated              bool
		method                     string
		route                      string
		body                       string
		expectedStatus             int
		expectedLocation           string
		expectedCacheControlHeader string
		hasCSPSandboxHeader        bool
		expectedBody               string
	}{
		{
			explanation:                "unauthenticated GET of a protected entry redirects to the unlock page",
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP/unlock",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation:                "unauthenticated GET of a protected entry with a filename redirects to the unlock page",
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/protected.txt",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP/unlock",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation:                "unauthenticated GET of the unlock page renders the challenge form",
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Protected Download",
		},
		{
			explanation:                "incorrect passphrase re-renders the challenge with 401",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			body:                       "passphrase=wrong+passphrase",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation:                "correct passphrase serves the file with sandbox CSP",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			body:                       "passphrase=correct+horse+battery+staple",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			hasCSPSandboxHeader:        true,
			expectedBody:               protectedEntry.Contents,
		},
		{
			explanation:                "passphrase only in the query string does not unlock the entry",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock?passphrase=correct+horse+battery+staple",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation:                "missing passphrase re-renders the challenge with 401",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation:                "oversized POST body is rejected",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			body:                       "passphrase=" + strings.Repeat("a", 4096),
			expectedStatus:             http.StatusBadRequest,
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation:                "malformed form body is rejected",
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			body:                       "passphrase=%zz",
			expectedStatus:             http.StatusBadRequest,
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation:    "POST to the download route is not allowed",
			authenticated:  false,
			method:         http.MethodPost,
			route:          "/-PPPPPPPPPP",
			body:           "passphrase=correct+horse+battery+staple",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			explanation:                "authenticated owner downloads a protected entry without a challenge",
			authenticated:              true,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			hasCSPSandboxHeader:        true,
			expectedBody:               protectedEntry.Contents,
		},
		{
			explanation:                "authenticated owner visiting the unlock page redirects to the download",
			authenticated:              true,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation:                "unlock page for an unprotected entry redirects to the download",
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-UUUUUUUUUU/unlock",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-UUUUUUUUUU",
			expectedCacheControlHeader: "no-store",
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
			if err := dataStore.InsertEntry(strings.NewReader(protectedEntry.Contents), picoshare.UploadMetadata{
				ID:                 protectedEntry.ID,
				Filename:           "protected.txt",
				ContentType:        "text/plain",
				Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
				Expires:            picoshare.NeverExpire,
				Size:               mustParseFileSize(len(protectedEntry.Contents)),
				DownloadPassphrase: protectedEntry.DownloadPassphrase,
			}); err != nil {
				t.Fatalf("failed to insert protected entry: %v", err)
			}
			if err := dataStore.InsertEntry(strings.NewReader(unprotectedEntry.Contents), picoshare.UploadMetadata{
				ID:          unprotectedEntry.ID,
				Filename:    "unprotected.txt",
				ContentType: "text/plain",
				Uploaded:    mustParseTime("2023-01-01T00:00:00Z"),
				Expires:     picoshare.NeverExpire,
				Size:        mustParseFileSize(len(unprotectedEntry.Contents)),
			}); err != nil {
				t.Fatalf("failed to insert unprotected entry: %v", err)
			}

			var authenticator handlers.Authenticator = unauthenticatedAuthenticator{}
			if tt.authenticated {
				authenticator = mockAuthenticator{}
			}
			s := handlers.New(authenticator, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			req := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.body))
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
			if got, want := res.Header.Get("Cache-Control"), tt.expectedCacheControlHeader; got != want {
				t.Errorf("Cache-Control=%q, want=%q", got, want)
			}
			if got, want := res.Header.Get("Content-Security-Policy") == "sandbox", tt.hasCSPSandboxHeader; got != want {
				t.Errorf("sandboxed CSP=%v, want=%v (Content-Security-Policy=%q)", got, want, res.Header.Get("Content-Security-Policy"))
			}
			if got, want := res.Header.Get("Location"), tt.expectedLocation; got != want {
				t.Errorf("Location=%q, want=%q", got, want)
			}
			if got := rec.Body.String(); !strings.Contains(got, tt.expectedBody) {
				t.Errorf("body=%q, want to contain %q", got, tt.expectedBody)
			}
		})
	}
}

func TestProtectedEntryDownloadRequiresPasswordEveryDownload(t *testing.T) {
	dataStore := test_sqlite.New(t)
	data := "protected file contents"
	passphrase := mustCreateDownloadPassphrase(t, "correct horse battery staple")
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:                 "PPPPPPPPPP",
		Filename:           "protected.txt",
		ContentType:        "text/plain",
		Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
		Expires:            picoshare.NeverExpire,
		Size:               mustParseFileSize(len(data)),
		DownloadPassphrase: passphrase,
	}); err != nil {
		t.Fatalf("failed to insert protected entry: %v", err)
	}
	s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)
	server := httptest.NewServer(s.Router())
	defer server.Close()

	// Use a client that retains cookies across requests, like a browser would,
	// so that any cookie the server set after a successful unlock would be sent
	// on the follow-up request.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{"passphrase": {"correct horse battery staple"}}
	res, err := client.PostForm(server.URL+"/-PPPPPPPPPP/unlock", form)
	if err != nil {
		t.Fatalf("failed to POST passphrase: %v", err)
	}
	defer res.Body.Close()
	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if got, want := string(body), data; got != want {
		t.Fatalf("body=%q, want=%q", got, want)
	}

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	if got := jar.Cookies(serverURL); len(got) != 0 {
		t.Errorf("cookies after unlock=%v, want none", got)
	}

	// On subsequent downloads, still require the passphrase to download, even
	// though the client sends back any cookies it received.
	followUpRes, err := client.Get(server.URL + "/-PPPPPPPPPP")
	if err != nil {
		t.Fatalf("failed to GET protected entry: %v", err)
	}
	defer followUpRes.Body.Close()
	if got, want := followUpRes.StatusCode, http.StatusFound; got != want {
		t.Errorf("follow-up status=%d, want=%d", got, want)
	}
	if got, want := followUpRes.Header.Get("Location"), "/-PPPPPPPPPP/unlock"; got != want {
		t.Errorf("follow-up Location=%q, want=%q", got, want)
	}
}

func mustCreateDownloadPassphrase(t *testing.T, value string) picoshare.DownloadPassphrase {
	t.Helper()

	passphrase, err := picoshare.NewDownloadPassphrase(value)
	if err != nil {
		t.Fatalf("failed to create download passphrase: %v", err)
	}

	return passphrase
}
