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
	dataStore := test_sqlite.New(t)
	passphrase, err := picoshare.NewPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create passphrase: %v", err)
	}
	hash, err := picoshare.HashDownloadPassphrase(passphrase)
	if err != nil {
		t.Fatalf("failed to hash passphrase: %v", err)
	}
	data := "protected file contents"
	metadata := picoshare.UploadMetadata{
		ID:                     "PPPPPPPPPP",
		Filename:               "protected.txt",
		ContentType:            "text/plain",
		Uploaded:               mustParseTime("2023-01-01T00:00:00Z"),
		Expires:                picoshare.NeverExpire,
		Size:                   mustParseFileSize(len(data)),
		DownloadPassphraseHash: &hash,
	}
	if err := dataStore.InsertEntry(strings.NewReader(data), metadata); err != nil {
		t.Fatalf("failed to insert protected entry: %v", err)
	}

	t.Run("unauthenticated GET renders a no-store challenge with nonce CSP", func(t *testing.T) {
		s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())
		req := httptest.NewRequest(http.MethodGet, "/-PPPPPPPPPP", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got := rec.Header().Get("Content-Security-Policy"); got == "sandbox" || !strings.Contains(got, "'nonce-") {
			t.Errorf("Content-Security-Policy=%q, want nonce policy", got)
		}
		if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
			t.Errorf("Cache-Control=%q, want=%q", got, want)
		}
		if got := rec.Body.String(); !strings.Contains(got, "Download passphrase") {
			t.Errorf("challenge body=%q, want passphrase form", got)
		}
	})

	t.Run("incorrect POST returns the challenge without authorizing later requests", func(t *testing.T) {
		s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())
		form := url.Values{"passphrase": {"wrong passphrase"}}
		req := httptest.NewRequest(http.MethodPost, "/-PPPPPPPPPP", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Errorf("status=%d, want=%d", got, want)
		}
		if got := rec.Header().Get("Set-Cookie"); got != "" {
			t.Errorf("Set-Cookie=%q, want empty", got)
		}
	})

	t.Run("correct POST serves content once with sandbox CSP and no-store", func(t *testing.T) {
		s := handlers.New(unauthenticatedAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())
		form := url.Values{"passphrase": {"correct horse battery staple"}}
		req := httptest.NewRequest(http.MethodPost, "/-PPPPPPPPPP", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Header().Get("Content-Security-Policy"), "sandbox"; got != want {
			t.Errorf("Content-Security-Policy=%q, want=%q", got, want)
		}
		if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
			t.Errorf("Cache-Control=%q, want=%q", got, want)
		}
		if got, want := rec.Body.String(), data; got != want {
			t.Errorf("body=%q, want=%q", got, want)
		}

		followUpReq := httptest.NewRequest(http.MethodGet, "/-PPPPPPPPPP", nil)
		followUpRec := httptest.NewRecorder()
		s.Router().ServeHTTP(followUpRec, followUpReq)
		if got := followUpRec.Body.String(); !strings.Contains(got, "Download passphrase") {
			t.Errorf("follow-up body=%q, want passphrase challenge", got)
		}
	})

	t.Run("authenticated owner bypasses the challenge", func(t *testing.T) {
		s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())
		req := httptest.NewRequest(http.MethodGet, "/-PPPPPPPPPP", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Body.String(), data; got != want {
			t.Errorf("body=%q, want=%q", got, want)
		}
		if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
			t.Errorf("Cache-Control=%q, want=%q", got, want)
		}
		if got, want := rec.Header().Get("Content-Security-Policy"), "sandbox"; got != want {
			t.Errorf("Content-Security-Policy=%q, want=%q", got, want)
		}
	})
}
