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
	ID          picoshare.EntryID
	Filename    picoshare.Filename
	ContentType picoshare.ContentType
}

type downloadTestAuthenticator struct{}

func (downloadTestAuthenticator) StartSession(http.ResponseWriter, *http.Request) {}

func (downloadTestAuthenticator) ClearSession(http.ResponseWriter) {}

func (downloadTestAuthenticator) Authenticate(r *http.Request) bool {
	return r.Header.Get("X-Test-Authenticated") == "true"
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

func TestEntryUnlock(t *testing.T) {
	for _, tt := range []struct {
		explanation      string
		method           string
		passphrase       string
		authenticated    bool
		expectedStatus   int
		expectedBody     string
		expectedLocation string
	}{
		{
			explanation:    "GET displays the passphrase form for a protected entry",
			method:         http.MethodGet,
			passphrase:     "",
			expectedStatus: http.StatusOK,
			expectedBody:   "Protected Download",
		},
		{
			explanation:    "POST with the correct passphrase downloads the protected entry",
			method:         http.MethodPost,
			passphrase:     "correct horse battery staple",
			expectedStatus: http.StatusOK,
			expectedBody:   "protected data",
		},
		{
			explanation:    "POST with an incorrect passphrase displays an authorization error",
			method:         http.MethodPost,
			passphrase:     "incorrect passphrase",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Incorrect passphrase.",
		},
		{
			explanation:      "GET redirects an authenticated requester to the download",
			method:           http.MethodGet,
			authenticated:    true,
			expectedStatus:   http.StatusFound,
			expectedLocation: "/-TTTTTTTTTT",
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			data := "protected data"
			entry := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:       dummyTextEntry.ID,
					Filename: dummyTextEntry.Filename,
					Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
					Expires:  picoshare.NeverExpire,
					Size:     mustParseFileSize(len(data)),
				},
				Reader: strings.NewReader(data),
			}
			if err := dataStore.InsertEntry(entry.Reader, entry.UploadMetadata); err != nil {
				t.Fatalf("failed to insert protected entry: %v", err)
			}

			s := handlers.New(downloadTestAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			updateRequest := httptest.NewRequest(
				http.MethodPut,
				"/api/entry/TTTTTTTTTT",
				strings.NewReader(`{"filename":"test.txt","downloadPassphrase":"correct horse battery staple"}`),
			)
			updateRequest.Header.Set("Content-Type", "application/json")
			updateRequest.Header.Set("X-Test-Authenticated", "true")
			updateRecorder := httptest.NewRecorder()
			s.Router().ServeHTTP(updateRecorder, updateRequest)
			if got, want := updateRecorder.Code, http.StatusOK; got != want {
				t.Fatalf("protected entry update status=%d, want=%d", got, want)
			}

			form := url.Values{"passphrase": {tt.passphrase}}
			req := httptest.NewRequest(tt.method, "/-TTTTTTTTTT/unlock", strings.NewReader(form.Encode()))
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if tt.authenticated {
				req.Header.Set("X-Test-Authenticated", "true")
			}
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.expectedStatus; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}
			if got, want := rec.Body.String(), tt.expectedBody; tt.expectedBody != "" && !strings.Contains(got, want) {
				t.Errorf("response body does not contain %q", want)
			}
			if got, want := res.Header.Get("Location"), tt.expectedLocation; got != want {
				t.Errorf("Location=%q, want=%q", got, want)
			}
		})
	}
}

func TestEntryUnlockPostParsesBoundedFormBody(t *testing.T) {
	dataStore := test_sqlite.New(t)
	data := "protected data"
	entry := picoshare.UploadEntry{
		UploadMetadata: picoshare.UploadMetadata{
			ID:       dummyTextEntry.ID,
			Filename: dummyTextEntry.Filename,
			Uploaded: mustParseTime("2023-01-01T00:00:00Z"),
			Expires:  picoshare.NeverExpire,
			Size:     mustParseFileSize(len(data)),
		},
		Reader: strings.NewReader(data),
	}
	if err := dataStore.InsertEntry(entry.Reader, entry.UploadMetadata); err != nil {
		t.Fatalf("failed to insert protected entry: %v", err)
	}

	now := mustParseTime("2023-01-01T00:00:00Z")
	s := handlers.New(downloadTestAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })

	updateRequest := httptest.NewRequest(
		http.MethodPut,
		"/api/entry/TTTTTTTTTT",
		strings.NewReader(`{"filename":"test.txt","downloadPassphrase":"correct horse battery staple"}`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set("X-Test-Authenticated", "true")
	updateRecorder := httptest.NewRecorder()
	s.Router().ServeHTTP(updateRecorder, updateRequest)
	if got, want := updateRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("protected entry update status=%d, want=%d", got, want)
	}

	for _, tt := range []struct {
		explanation    string
		route          string
		body           string
		expectedStatus int
	}{
		{
			explanation:    "passphrase only in query string does not unlock the entry",
			route:          "/-TTTTTTTTTT/unlock?passphrase=correct+horse+battery+staple",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			explanation:    "oversized POST body is rejected",
			route:          "/-TTTTTTTTTT/unlock",
			body:           "passphrase=" + strings.Repeat("a", 4096),
			expectedStatus: http.StatusBadRequest,
		},
		{
			explanation:    "malformed form body is rejected",
			route:          "/-TTTTTTTTTT/unlock",
			body:           "passphrase=%zz",
			expectedStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.route, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.Router().ServeHTTP(rec, req)

			if got, want := rec.Code, tt.expectedStatus; got != want {
				t.Errorf("status=%d, want=%d", got, want)
			}
		})
	}
}
