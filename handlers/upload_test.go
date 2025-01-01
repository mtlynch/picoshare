package handlers_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
}

type mockClock struct {
	t time.Time
}

func (c mockClock) Now() time.Time {
	return c.t
}

func TestEntryPost(t *testing.T) {
	for _, tt := range []struct {
		description string
		filename    string
		contents    string
		expiration  string
		note        string
		status      int
	}{
		{
			description: "valid file with no note",
			filename:    "dummyimage.png",
			contents:    "dummy bytes",
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusOK,
		},
		{
			description: "valid file with a note",
			filename:    "dummyimage.png",
			contents:    "dummy bytes",
			note:        "for my homeboy, willy",
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusOK,
		},
		{
			description: "valid file with a too-long note",
			filename:    "dummyimage.png",
			contents:    "dummy bytes",
			note:        strings.Repeat("A", parse.MaxFileNoteBytes+1),
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusBadRequest,
		},
		{
			description: "filename that's just a dot",
			filename:    ".",
			contents:    "dummy bytes",
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusBadRequest,
		},
		{
			description: "empty upload",
			filename:    "dummy.png",
			contents:    "",
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusBadRequest,
		},
		{
			description: "expiration in the past",
			filename:    "dummy.png",
			contents:    "dummy bytes",
			expiration:  "2000-01-01T00:00:00Z",
			status:      http.StatusBadRequest,
		},
		{
			description: "invalid expiration",
			filename:    "dummy.png",
			contents:    "dummy bytes",
			expiration:  "invalid-expiration-date",
			status:      http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

			formData, contentType := createMultipartFormBody(tt.filename, tt.note, bytes.NewBuffer([]byte(tt.contents)))

			req, err := http.NewRequest("POST", "/api/entry?expiration="+tt.expiration, formData)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", contentType)

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Errorf("status=%d, want=%d", got, want)
			}

			// Only check the response if the request succeeded.
			if res.StatusCode != http.StatusOK {
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read response body")
			}

			var response handlers.EntryPostResponse
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatalf("response is not valid JSON: %v", body)
			}

			entry, err := dataStore.GetEntry(picoshare.EntryID(response.ID))
			if err != nil {
				t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
			}

			if got, want := mustReadAll(entry.Reader), []byte(tt.contents); !reflect.DeepEqual(got, want) {
				t.Errorf("stored contents= %v, want=%v", got, want)
			}

			if got, want := entry.Filename, picoshare.Filename(tt.filename); got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}

			if got, want := entry.Expires, mustParseExpirationTime(tt.expiration); got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}
		})
	}
}

func TestEntryPut(t *testing.T) {
	originalEntry := picoshare.UploadMetadata{
		ID:       picoshare.EntryID("AAAAAAAAAA"),
		Filename: picoshare.Filename("original-filename.mp3"),
		Expires:  mustParseExpirationTime("2024-12-15T21:52:33Z"),
		Note:     picoshare.FileNote{},
	}
	for _, tt := range []struct {
		description      string
		targetID         string
		payload          string
		filenameExpected string
		expiresExpected  picoshare.ExpirationTime
		noteExpected     picoshare.FileNote
		status           int
	}{
		{
			description: "updates metadata for valid request",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track"
			}`,
			filenameExpected: "cool-song.mp3",
			noteExpected:     makeNote("My latest track"),
			expiresExpected:  mustParseExpirationTime("2029-01-02T01:02:03Z"),
			status:           http.StatusOK,
		},
		{
			description: "treats missing expiration time as NeverExpire",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"note":"My latest track"
			}`,
			filenameExpected: "cool-song.mp3",
			noteExpected:     makeNote("My latest track"),
			expiresExpected:  picoshare.NeverExpire,
			status:           http.StatusOK,
		},
		{
			description: "rejects update when filename is invalid",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track"
			}`,
			filenameExpected: "original-filename.mp3",
			noteExpected:     picoshare.FileNote{},
			expiresExpected:  mustParseExpirationTime("2024-12-15T21:52:33Z"),
			status:           http.StatusBadRequest,
		},
		{
			description: "rejects update when note is invalid",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"<script>alert(1)</script>"
			}`,
			filenameExpected: "original-filename.mp3",
			expiresExpected:  mustParseExpirationTime("2024-12-15T21:52:33Z"),
			noteExpected:     picoshare.FileNote{},
			status:           http.StatusBadRequest,
		},
		{
			description: "ignores non-existent entry ID",
			targetID:    "BBBBBBBBBB",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track"
			}`,
			filenameExpected: "original-filename.mp3",
			expiresExpected:  mustParseExpirationTime("2024-12-15T21:52:33Z"),
			noteExpected:     picoshare.FileNote{},
			status:           http.StatusNotFound,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			store := test_sqlite.New()
			originalData := "dummy original data"
			metadata := originalEntry
			metadata.Size = mustParseFileSize(len(originalData))
			store.InsertEntry(strings.NewReader((originalData)), metadata)
			s := handlers.New(mockAuthenticator{}, &store, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

			req, err := http.NewRequest("PUT", "/api/entry/"+tt.targetID, strings.NewReader(tt.payload))
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

			entry, err := store.GetEntry(picoshare.EntryID(originalEntry.ID))
			if err != nil {
				t.Fatalf("failed to get expected entry %v from data store: %v", originalEntry.ID, err)
			}

			if got, want := entry.Filename, picoshare.Filename(tt.filenameExpected); got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}

			if got, want := entry.Note.String(), tt.noteExpected.String(); got != want {
				t.Errorf("note=%v, want=%v", got, want)
			}
		})
	}
}

func TestGuestUpload(t *testing.T) {
	authenticator, err := shared_secret.New("dummypass")
	if err != nil {
		t.Fatalf("failed to create shared secret: %v", err)
	}

	for _, tt := range []struct {
		description                string
		guestLinkInStore           picoshare.GuestLink
		entriesInStore             []picoshare.UploadEntry
		currentTime                time.Time
		guestLinkID                string
		note                       string
		status                     int
		fileExpirationTimeExpected picoshare.ExpirationTime
	}{
		{
			description: "valid upload to guest link whose files never expire",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "expired guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2000-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusUnauthorized,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "invalid guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "i-am-an-invalid-guest-link", // Too long
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "invalid guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "I0OI0OI0OI0OI0OI", // Contains all invalid characters
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "non-existent guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2000-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "doesntexistaaaaa",
			status:                     http.StatusNotFound,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "reject upload that includes a note",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			note:                       "I'm a disallowed note",
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "exhausted upload count",
			guestLinkInStore: picoshare.GuestLink{
				ID:             picoshare.GuestLinkID("abcdefgh23456789"),
				Created:        mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:     mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileUploads: makeGuestUploadCountLimit(2),
				FileLifetime:   picoshare.FileLifetimeInfinite,
			},
			entriesInStore: []picoshare.UploadEntry{
				{
					UploadMetadata: picoshare.UploadMetadata{
						ID: picoshare.EntryID("dummy-entry1"),
						GuestLink: picoshare.GuestLink{
							ID: picoshare.GuestLinkID("abcdefgh23456789"),
						},
					},
				},
				{
					UploadMetadata: picoshare.UploadMetadata{
						ID: picoshare.EntryID("dummy-entry2"),
						GuestLink: picoshare.GuestLink{
							ID: picoshare.GuestLinkID("abcdefgh23456789"),
						},
					},
				},
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusUnauthorized,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "exhausted upload bytes",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes: makeGuestUploadMaxFileBytes(1),
				FileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "guest file expires in 1 day",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInDays(1),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-02T00:00:00Z"),
		},
		{
			description: "guest file expires in 365 days",
			guestLinkInStore: picoshare.GuestLink{
				ID:           picoshare.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2022-01-01T00:00:00Z"),
				UrlExpires:   mustParseExpirationTime("2030-01-02T03:04:25Z"),
				FileLifetime: picoshare.NewFileLifetimeInDays(365),
			},
			currentTime:                mustParseTime("2023-01-01T00:00:00Z"),
			guestLinkID:                "abcdefgh23456789",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-01T00:00:00Z"),
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			store := test_sqlite.New()
			if err := store.InsertGuestLink(tt.guestLinkInStore); err != nil {
				t.Fatalf("failed to insert dummy guest link: %v", err)
			}
			for _, entry := range tt.entriesInStore {
				data := "dummy data"
				entry.UploadMetadata.Size = mustParseFileSize(len(data))
				if err := store.InsertEntry(strings.NewReader(data), entry.UploadMetadata); err != nil {
					t.Fatalf("failed to insert dummy entry: %v", err)
				}
			}

			c := mockClock{tt.currentTime}
			s := handlers.New(authenticator, &store, nilSpaceChecker, nilGarbageCollector, c)

			filename := "dummyimage.png"
			contents := "dummy bytes"
			formData, contentType := createMultipartFormBody(filename, tt.note, strings.NewReader(contents))

			req, err := http.NewRequest("POST", "/api/guest/"+tt.guestLinkID, formData)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", contentType)
			req.Header.Add("Accept", "application/json")

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			// Only check the response if the request succeeded.
			if res.StatusCode != http.StatusOK {
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read response body")
			}

			var response handlers.EntryPostResponse
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatalf("response is not valid JSON: %v", body)
			}

			entry, err := store.GetEntry(picoshare.EntryID(response.ID))
			if err != nil {
				t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
			}

			if got, want := mustReadAll(entry.Reader), []byte(contents); !reflect.DeepEqual(got, want) {
				t.Errorf("stored contents= %v, want=%v", got, want)
			}

			if got, want := entry.Filename, picoshare.Filename(filename); got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}

			if got, want := entry.Expires, tt.fileExpirationTimeExpected; got != want {
				t.Errorf("file expiration=%v, want=%v", got, want)
			}
		})
	}
}

func createMultipartFormBody(filename, note string, r io.Reader) (io.Reader, string) {
	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)

	f, err := mw.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	io.Copy(f, r)

	nf, err := mw.CreateFormField("note")
	if err != nil {
		panic(err)
	}
	nf.Write([]byte(note))

	mw.Close()
	bw.Flush()

	return bufio.NewReader(&b), mw.FormDataContentType()
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func mustParseExpirationTime(s string) picoshare.ExpirationTime {
	return picoshare.ExpirationTime(mustParseTime(s))
}

func mustReadAll(r io.Reader) []byte {
	d, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return d
}

func makeNote(s string) picoshare.FileNote {
	return picoshare.FileNote{Value: &s}
}

func mustParseFileSize(val int) picoshare.FileSize {
	fileSize, err := picoshare.FileSizeFromInt(val)
	if err != nil {
		panic(err)
	}

	return fileSize
}
