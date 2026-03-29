package handlers_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
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
			expiration:  "2024-01-01T00:00:00Z",
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

			entry, err := dataStore.GetEntryMetadata(picoshare.EntryID(response.ID))
			if err != nil {
				t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
			}

			if got, want := entry.Filename, picoshare.Filename(tt.filename); got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}

			if got, want := entry.Expires, mustParseExpirationTime(tt.expiration); got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}

			entryFile, err := dataStore.ReadEntryFile(entry.ID)
			if err != nil {
				t.Fatalf("failed to read file for entry %v: %v", entry.ID, err)
			}
			if got, want := mustReadAll(entryFile), []byte(tt.contents); !reflect.DeepEqual(got, want) {
				t.Errorf("stored contents= %v, want=%v", got, want)
			}

		})
	}
}

func TestEntryPut(t *testing.T) {
	originalEntry := picoshare.UploadMetadata{
		ID:          picoshare.EntryID("AAAAAAAAAA"),
		Filename:    picoshare.Filename("original-filename.mp3"),
		ContentType: picoshare.ContentType("audio/mpeg"),
		Uploaded:    mustParseTime("2023-01-01T00:00:00Z"),
		Expires:     mustParseExpirationTime("2024-12-15T21:52:33Z"),
		Note:        picoshare.FileNote{},
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
			dataStore := test_sqlite.New()
			originalData := "dummy original data"
			metadata := originalEntry
			metadata.Size = mustParseFileSize(len(originalData))
			dataStore.InsertEntry(strings.NewReader((originalData)), metadata)
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock())

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

			entry, err := dataStore.GetEntryMetadata(picoshare.EntryID(originalEntry.ID))
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
		url                        string
		note                       string
		status                     int
		fileExpirationTimeExpected picoshare.ExpirationTime
	}{
		{
			description: "valid upload to guest link whose files never expire",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2030-01-01T00:00:00Z"),
		},
		{
			description: "expired guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2024-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			status:                     http.StatusUnauthorized,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "invalid guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/i-am-an-invalid-guest-link?expiration=2030-01-01T00:00:00Z", // Too long
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "invalid guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/I0OI0OI0OI0OI0OI?expiration=2030-01-01T00:00:00Z", // Contains all invalid characters
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "non-existent guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2024-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/doesntexistaaaaa?expiration=2030-01-01T00:00:00Z",
			status:                     http.StatusNotFound,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "reject upload that includes a note",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			note:                       "I'm a disallowed note",
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "exhausted upload count",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileUploads:  makeGuestUploadCountLimit(2),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			entriesInStore: []picoshare.UploadEntry{
				{
					UploadMetadata: picoshare.UploadMetadata{
						ID:       picoshare.EntryID("dummy-entry1"),
						Uploaded: mustParseTime("2024-02-01T00:00:00Z"),
						GuestLink: picoshare.GuestLink{
							ID: picoshare.GuestLinkID("abcdefgh23456789"),
						},
						Expires: picoshare.NeverExpire,
					},
				},
				{
					UploadMetadata: picoshare.UploadMetadata{
						ID:       picoshare.EntryID("dummy-entry2"),
						Uploaded: mustParseTime("2024-02-02T00:00:00Z"),
						GuestLink: picoshare.GuestLink{
							ID: picoshare.GuestLinkID("abcdefgh23456789"),
						},
						Expires: picoshare.NeverExpire,
					},
				},
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			status:                     http.StatusUnauthorized,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "exhausted upload bytes",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "guest file expires in 1 day",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(1),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2024-01-02T00:00:00Z",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-02T00:00:00Z"),
		},
		{
			description: "guest file expires in 365 days",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(365),
			},
			currentTime:                mustParseTime("2023-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2024-01-01T00:00:00Z",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-01T00:00:00Z"),
		},
		{
			description: "guest upload with valid expiration within guest link limits",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(30),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2024-01-15T00:00:00Z",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-15T00:00:00Z"),
		},
		{
			description: "guest upload with infinite guest link accepts any expiration",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2025-01-01T00:00:00Z",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2025-01-01T00:00:00Z"),
		},
		{
			description: "reject guest upload with expiration that exceeds guest link limit",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(7),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2024-01-31T00:00:00Z",
			status:                     http.StatusBadRequest,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-08T00:00:00Z"),
		},
		{
			description: "guest upload without expiration defaults to max allowed (30 days)",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(30),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-31T00:00:00Z"),
		},
		{
			description: "guest upload with empty expiration defaults to max allowed (30 days)",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.NewFileLifetimeInDays(30),
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: mustParseExpirationTime("2024-01-31T00:00:00Z"),
		},
		{
			description: "guest upload without expiration defaults to never expire (infinite)",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
		{
			description: "guest upload with empty expiration defaults to never expire (infinite)",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=",
			status:                     http.StatusOK,
			fileExpirationTimeExpected: picoshare.NeverExpire,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New()
			if err := dataStore.InsertGuestLink(tt.guestLinkInStore); err != nil {
				t.Fatalf("failed to insert dummy guest link: %v", err)
			}
			for _, entry := range tt.entriesInStore {
				data := "dummy data"
				entry.UploadMetadata.Size = mustParseFileSize(len(data))
				if err := dataStore.InsertEntry(strings.NewReader(data), entry.UploadMetadata); err != nil {
					t.Fatalf("failed to insert dummy entry: %v", err)
				}
			}

			c := mockClock{tt.currentTime}
			s := handlers.New(authenticator, &dataStore, nilSpaceChecker, nilGarbageCollector, c)

			filename := "dummyimage.png"
			contents := "dummy bytes"
			formData, contentType := createMultipartFormBody(filename, tt.note, strings.NewReader(contents))

			req, err := http.NewRequest("POST", tt.url, formData)
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

			entry, err := dataStore.GetEntryMetadata(picoshare.EntryID(response.ID))
			if err != nil {
				t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
			}

			if got, want := entry.Filename, picoshare.Filename(filename); got != want {
				t.Errorf("filename=%v, want=%v", got, want)
			}

			if got, want := entry.Expires, tt.fileExpirationTimeExpected; got != want {
				t.Errorf("file expiration=%v, want=%v", got, want)
			}

			entryFile, err := dataStore.ReadEntryFile(entry.ID)
			if err != nil {
				t.Fatalf("failed to read entry file for %v: %v", entry.ID, err)
			}
			if got, want := mustReadAll(entryFile), []byte(contents); !reflect.DeepEqual(got, want) {
				t.Errorf("stored contents= %v, want=%v", got, want)
			}
		})
	}
}

func TestGuestUploadAcceptHeader(t *testing.T) {
	authenticator, err := shared_secret.New("dummypass")
	if err != nil {
		t.Fatalf("failed to create shared secret: %v", err)
	}

	for _, tt := range []struct {
		explanation         string
		acceptHeader        string
		expectJSON          bool
		expectedContentType string
	}{
		{
			"no Accept header returns plain text URL",
			"",
			false,
			"text/plain",
		},
		{
			"Accept header with wildcard returns plain text URL",
			"*/*",
			false,
			"text/plain",
		},
		{
			"Accept header with application/json returns JSON",
			"application/json",
			true,
			"application/json",
		},
		{
			"Accept header with text/html returns plain text URL",
			"text/html",
			false,
			"text/plain",
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.explanation, tt.acceptHeader), func(t *testing.T) {
			dataStore := test_sqlite.New()
			guestLink := picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			}
			if err := dataStore.InsertGuestLink(guestLink); err != nil {
				t.Fatalf("failed to insert dummy guest link: %v", err)
			}

			c := mockClock{mustParseTime("2024-01-01T00:00:00Z")}
			s := handlers.New(authenticator, &dataStore, nilSpaceChecker, nilGarbageCollector, c)

			filename := "dummyimage.png"
			contents := "dummy bytes"
			formData, contentType := createMultipartFormBody(filename, "", strings.NewReader(contents))

			req, err := http.NewRequest("POST", "/api/guest/abcdefgh23456789", formData)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("Content-Type", contentType)
			if tt.acceptHeader != "" {
				req.Header.Add("Accept", tt.acceptHeader)
			}

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, http.StatusOK; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			if got, want := res.Header.Get("Content-Type"), tt.expectedContentType; got != want {
				t.Errorf("Content-Type=%v, want=%v", got, want)
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("failed to read response body")
			}

			if tt.expectJSON {
				var response handlers.EntryPostResponse
				err = json.Unmarshal(body, &response)
				if err != nil {
					t.Fatalf("response is not valid JSON: %v", string(body))
				}
				if got, want := len(response.ID), 10; got != want {
					t.Errorf("ID length=%d, want=%d", got, want)
				}
			} else {
				// Should be plain text URL.
				bodyStr := string(body)
				if !strings.Contains(bodyStr, "http") {
					t.Errorf("expected URL in response, got: %v", bodyStr)
				}
				if !strings.HasSuffix(bodyStr, "\r\n") {
					t.Errorf("expected response to end with \\r\\n, got: %v", bodyStr)
				}
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
