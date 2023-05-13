package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

func TestEntryGet(t *testing.T) {
	for _, tt := range []struct {
		description                string
		requestRoute               string
		expectedStatus             int
		expectedContentDisposition string
		expectedContentType        string
	}{
		{
			description:                "retrieves text entry",
			requestRoute:               "/-TTTTTTTTTT",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.txt"`,
			expectedContentType:        "text/plain;charset=utf-8",
		},
		{
			description:                "retrieves audio entry",
			requestRoute:               "/-AAAAAAAAAA",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.mp3"`,
			expectedContentType:        "audio/mpeg",
		},
		{
			description:                "retrieves audio entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-AAAAAAAA22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp3"`,
			expectedContentType:        "audio/mpeg",
		},
		{
			description:                "retrieves video entry",
			requestRoute:               "/-VVVVVVVVVV",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test.mp4"`,
			expectedContentType:        "video/mp4",
		},
		{
			description:                "retrieves video entry and infers content-type when it wasn't specified at upload time",
			requestRoute:               "/-VVVVVVVV22",
			expectedStatus:             http.StatusOK,
			expectedContentDisposition: `filename="test0.mp4"`,
			expectedContentType:        "video/mp4",
		},
		{
			description:    "request for non-existent entry returns 404",
			requestRoute:   "/-ZZZZZZZZZZ",
			expectedStatus: http.StatusNotFound,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()

			// We have to create the dummy data in each test, otherwise the state of
			// the Reader property will change across tests.
			dummyTextEntry := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:          "TTTTTTTTTT",
					Filename:    picoshare.Filename("test.txt"),
					ContentType: picoshare.ContentType("text/plain;charset=utf-8"),
				},
				Reader: strings.NewReader("dummy text contents"),
			}
			dummyAudioEntry := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:          "AAAAAAAAAA",
					Filename:    picoshare.Filename("test.mp3"),
					ContentType: picoshare.ContentType("audio/mpeg"),
				},
				Reader: strings.NewReader("dummy audio contents"),
			}
			dummyAudioEntrywithoutContentType := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:       "AAAAAAAA22",
					Filename: picoshare.Filename("test0.mp3"),
				},
				Reader: strings.NewReader("dummy audio contents"),
			}
			dummyVideoEntry := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:          "VVVVVVVVVV",
					Filename:    picoshare.Filename("test.mp4"),
					ContentType: picoshare.ContentType("video/mp4"),
				},
				Reader: strings.NewReader("dummy video contents"),
			}
			dummyVideoEntryWithoutContentType := picoshare.UploadEntry{
				UploadMetadata: picoshare.UploadMetadata{
					ID:       "VVVVVVVV22",
					Filename: picoshare.Filename("test0.mp4"),
				},
				Reader: strings.NewReader("dummy video contents"),
			}
			for _, entry := range []picoshare.UploadEntry{
				dummyTextEntry,
				dummyAudioEntry,
				dummyAudioEntrywithoutContentType,
				dummyVideoEntry,
				dummyVideoEntryWithoutContentType,
			} {
				if err := dataStore.InsertEntry(entry.Reader, entry.UploadMetadata); err != nil {
					panic(err)
				}
			}

			s, err := handlers.New(mockAuthenticator{}, dataStore, nilSpaceChecker, nilGarbageCollector)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest("GET", tt.requestRoute, nil)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			s.Router().ServeHTTP(w, req)

			if got, want := w.Code, tt.expectedStatus; got != want {
				t.Fatalf("%s returned wrong status code: got %v want %v",
					tt.requestRoute, got, want)
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			if got, want := w.Header().Get("Content-Disposition"), tt.expectedContentDisposition; got != want {
				t.Errorf("Content-Disposition=%s, want=%s", got, want)
			}

			if got, want := w.Header().Get("Content-Type"), tt.expectedContentType; got != want {
				t.Errorf("Content-Type=%s, want=%s", got, want)
			}
		})
	}
}
