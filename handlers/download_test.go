package handlers_test

import (
	"net/http"
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
	ID          string
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
	dummyVideoEntryWithContentTypeParameters = mockEntry{
		ID:          "VVVVVVVV33",
		Filename:    picoshare.Filename("test1.mp4"),
		ContentType: picoshare.ContentType(`video/mp4; codecs="vp09"`),
	}
	dummyHTMLEntryWithVideoFilename = mockEntry{
		ID:          "HHHHHHHHHH",
		Filename:    picoshare.Filename("payload.mp4"),
		ContentType: picoshare.ContentType("text/html"),
	}
	dummySVGEntry = mockEntry{
		ID:          "SSSSSSSSSS",
		Filename:    picoshare.Filename("payload.svg"),
		ContentType: picoshare.ContentType("image/svg+xml"),
	}
	dummyHTMLEntryWithMalformedMediaType = mockEntry{
		ID:          "MMMMMMMMMM",
		Filename:    picoshare.Filename("payload.html"),
		ContentType: picoshare.ContentType("video/mp4 text/html"),
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
			expectedCSP:                "sandbox allow-same-origin",
		},
		{
			description:                "retrieves audio entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-AAAAAAAA22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp3"`,
			expectedContentType:        "audio/mpeg",
			expectedCSP:                "sandbox allow-same-origin",
		},
		{
			description:                "retrieves video entry",
			requestRoute:               "/-VVVVVVVVVV",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.mp4"`,
			expectedContentType:        "video/mp4",
			expectedCSP:                "sandbox allow-same-origin",
		},
		{
			description:                "retrieves video entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-VVVVVVVV22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp4"`,
			expectedContentType:        "video/mp4",
			expectedCSP:                "sandbox allow-same-origin",
		},
		{
			description:                "retrieves video entry with content-type parameters",
			requestRoute:               "/-VVVVVVVV33",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test1.mp4"`,
			expectedContentType:        `video/mp4; codecs="vp09"`,
			expectedCSP:                "sandbox allow-same-origin",
		},
		{
			description:                "retrieves HTML entry with a video filename",
			requestRoute:               "/-HHHHHHHHHH",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="payload.mp4"`,
			expectedContentType:        "text/html",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves SVG entry",
			requestRoute:               "/-SSSSSSSSSS",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="payload.svg"`,
			expectedContentType:        "image/svg+xml",
			expectedCSP:                "sandbox",
		},
		{
			description:                "retrieves HTML entry with a malformed media type and keeps the strict sandbox",
			requestRoute:               "/-MMMMMMMMMM",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="payload.html"`,
			expectedContentType:        "video/mp4 text/html",
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
				dummyVideoEntryWithContentTypeParameters,
				dummyHTMLEntryWithVideoFilename,
				dummySVGEntry,
				dummyHTMLEntryWithMalformedMediaType,
			} {
				data := "dummy data"
				entry := picoshare.UploadEntry{
					UploadMetadata: picoshare.UploadMetadata{
						ID:          picoshare.MustCreateEntryID(mockEntry.ID),
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
		ID                 string
		Contents           string
		DownloadPassphrase picoshare.DownloadPassphrase
	}

	for _, tt := range []struct {
		explanation                string
		entryInStore               fakeEntry
		authenticated              bool
		method                     string
		route                      string
		passphrase                 string
		expectedStatus             int
		expectedLocation           string
		expectedCacheControlHeader string
		hasCSPSandboxHeader        bool
		expectedBody               string
	}{
		{
			explanation: "unauthenticated GET of a protected entry redirects to the unlock page",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP/unlock",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation: "unauthenticated GET of a protected entry with a filename redirects to the unlock page",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/protected.txt",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP/unlock",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation: "unauthenticated GET of the unlock page renders the challenge form",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Protected Download",
		},
		{
			explanation: "incorrect passphrase re-renders the challenge with 401",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			passphrase:                 "wrong passphrase",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation: "correct passphrase serves the file with sandbox CSP",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			passphrase:                 "correct horse battery staple",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			hasCSPSandboxHeader:        true,
			expectedBody:               "fake protected data",
		},
		{
			explanation: "passphrase only in the query string does not unlock the entry",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock?passphrase=correct+horse+battery+staple",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation: "missing passphrase re-renders the challenge with 401",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation: "oversized POST body is rejected",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			passphrase:                 strings.Repeat("a", 4096),
			expectedStatus:             http.StatusBadRequest,
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation: "passphrase with percent characters is rejected",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              false,
			method:                     http.MethodPost,
			route:                      "/-PPPPPPPPPP/unlock",
			passphrase:                 "%zz",
			expectedStatus:             http.StatusUnauthorized,
			expectedCacheControlHeader: "no-store",
			expectedBody:               "Incorrect passphrase.",
		},
		{
			explanation: "POST to the download route is not allowed",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:  false,
			method:         http.MethodPost,
			route:          "/-PPPPPPPPPP",
			passphrase:     "correct horse battery staple",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			explanation: "authenticated owner downloads a protected entry without a challenge",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              true,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP",
			expectedStatus:             http.StatusOK,
			expectedCacheControlHeader: "no-store",
			hasCSPSandboxHeader:        true,
			expectedBody:               "fake protected data",
		},
		{
			explanation: "authenticated owner visiting the unlock page redirects to the download",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:              true,
			method:                     http.MethodGet,
			route:                      "/-PPPPPPPPPP/unlock",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-PPPPPPPPPP",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation: "unlock page for an unprotected entry redirects to the download",
			entryInStore: fakeEntry{
				ID:       "UUUUUUUUUU",
				Contents: "fake unprotected data",
			},
			authenticated:              false,
			method:                     http.MethodGet,
			route:                      "/-UUUUUUUUUU/unlock",
			expectedStatus:             http.StatusFound,
			expectedLocation:           "/-UUUUUUUUUU",
			expectedCacheControlHeader: "no-store",
		},
		{
			explanation: "unlock page for a non-existent entry returns 404",
			entryInStore: fakeEntry{
				ID:                 "PPPPPPPPPP",
				Contents:           "fake protected data",
				DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			},
			authenticated:  false,
			method:         http.MethodGet,
			route:          "/-ZZZZZZZZZZ/unlock",
			expectedStatus: http.StatusNotFound,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			if err := dataStore.InsertEntry(strings.NewReader(tt.entryInStore.Contents), picoshare.UploadMetadata{
				ID:                 picoshare.MustCreateEntryID(tt.entryInStore.ID),
				Filename:           "test.txt",
				ContentType:        "text/plain",
				Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
				Expires:            picoshare.NeverExpire,
				Size:               mustParseFileSize(len(tt.entryInStore.Contents)),
				DownloadPassphrase: tt.entryInStore.DownloadPassphrase,
			}); err != nil {
				t.Fatalf("failed to insert entry: %v", err)
			}

			var authenticator handlers.Authenticator = unauthenticatedAuthenticator{}
			if tt.authenticated {
				authenticator = mockAuthenticator{}
			}
			s := handlers.New(authenticator, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			body := url.Values{"passphrase": {tt.passphrase}}.Encode()
			req := httptest.NewRequest(tt.method, tt.route, strings.NewReader(body))
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

func TestProtectedEntryDownloadRequiresPassphraseEveryDownload(t *testing.T) {
	dataStore := test_sqlite.New(t)
	data := "protected file contents"
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:                 picoshare.MustCreateEntryID("PPPPPPPPPP"),
		Filename:           "protected.txt",
		ContentType:        "text/plain",
		Uploaded:           mustParseTime("2023-01-01T00:00:00Z"),
		Expires:            picoshare.NeverExpire,
		Size:               mustParseFileSize(len(data)),
		DownloadPassphrase: mustCreateDownloadPassphrase(t, "correct horse battery staple"),
	}); err != nil {
		t.Fatalf("failed to insert protected entry: %v", err)
	}
	s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	{
		req := httptest.NewRequest(http.MethodPost, "/-PPPPPPPPPP/unlock", strings.NewReader("passphrase=correct+horse+battery+staple"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Body.String(), data; got != want {
			t.Fatalf("body=%q, want=%q", got, want)
		}
	}

	// A successful unlock must not let the same server serve the entry without
	// the passphrase on the next request.
	{
		req := httptest.NewRequest(http.MethodGet, "/-PPPPPPPPPP", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusFound; got != want {
			t.Errorf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Header().Get("Location"), "/-PPPPPPPPPP/unlock"; got != want {
			t.Errorf("Location=%q, want=%q", got, want)
		}
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
