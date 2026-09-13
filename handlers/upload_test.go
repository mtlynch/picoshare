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

func TestEntryPost(t *testing.T) {
	for _, tt := range []struct {
		description string
		filename    string
		contents    string
		expiration  string
		note        string
		passphrase  string
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
			description: "valid file with a download passphrase",
			filename:    "dummyimage.png",
			contents:    "dummy bytes",
			passphrase:  "correct horse battery staple",
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusOK,
		},
		{
			description: "invalid download passphrase is rejected",
			filename:    "dummyimage.png",
			contents:    "dummy bytes",
			passphrase:  strings.Repeat("a", picoshare.MaxPassphraseCodePoints+1),
			expiration:  "2040-01-01T00:00:00Z",
			status:      http.StatusBadRequest,
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
			dataStore := test_sqlite.New(t)
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			formData, contentType := createMultipartFormBody(tt.filename, tt.note, tt.passphrase, bytes.NewBuffer([]byte(tt.contents)))

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/entry?expiration="+tt.expiration,
				formData,
			)
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

			if got, want := entry.DownloadPassphrase.String(), tt.passphrase; got != want {
				t.Errorf("download passphrase=%q, want=%q", got, want)
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
		// passphraseInStore protects the original entry when non-empty.
		passphraseInStore string
		// passphraseExpected is the passphrase the entry must accept after the
		// request, or empty if the entry must be unprotected.
		passphraseExpected string
		status             int
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
		{
			description: "sets a download passphrase",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track",
				"downloadPassphrase": "correct horse battery staple"
			}`,
			filenameExpected:   "cool-song.mp3",
			noteExpected:       makeNote("My latest track"),
			expiresExpected:    mustParseExpirationTime("2029-01-02T01:02:03Z"),
			passphraseExpected: "correct horse battery staple",
			status:             http.StatusOK,
		},
		{
			description: "replaces an existing download passphrase",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track",
				"downloadPassphrase": "new passphrase"
			}`,
			filenameExpected:   "cool-song.mp3",
			noteExpected:       makeNote("My latest track"),
			expiresExpected:    mustParseExpirationTime("2029-01-02T01:02:03Z"),
			passphraseInStore:  "correct horse battery staple",
			passphraseExpected: "new passphrase",
			status:             http.StatusOK,
		},
		{
			description: "removes the download passphrase",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track",
				"removeDownloadPassphrase": true
			}`,
			filenameExpected:   "cool-song.mp3",
			noteExpected:       makeNote("My latest track"),
			expiresExpected:    mustParseExpirationTime("2029-01-02T01:02:03Z"),
			passphraseInStore:  "correct horse battery staple",
			passphraseExpected: "",
			status:             http.StatusOK,
		},
		{
			description: "keeps the download passphrase when the request omits it",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track"
			}`,
			filenameExpected:   "cool-song.mp3",
			noteExpected:       makeNote("My latest track"),
			expiresExpected:    mustParseExpirationTime("2029-01-02T01:02:03Z"),
			passphraseInStore:  "correct horse battery staple",
			passphraseExpected: "correct horse battery staple",
			status:             http.StatusOK,
		},
		{
			description: "rejects update when download passphrase is too long",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track",
				"downloadPassphrase": "` + strings.Repeat("a", picoshare.MaxPassphraseCodePoints+1) + `"
			}`,
			filenameExpected:   "original-filename.mp3",
			noteExpected:       picoshare.FileNote{},
			expiresExpected:    mustParseExpirationTime("2024-12-15T21:52:33Z"),
			passphraseInStore:  "correct horse battery staple",
			passphraseExpected: "correct horse battery staple",
			status:             http.StatusBadRequest,
		},
		{
			description: "rejects update that both sets and removes the download passphrase",
			targetID:    "AAAAAAAAAA",
			payload: `{
				"filename": "cool-song.mp3",
				"expiration": "2029-01-02T01:02:03Z",
				"note":"My latest track",
				"downloadPassphrase": "new passphrase",
				"removeDownloadPassphrase": true
			}`,
			filenameExpected:   "original-filename.mp3",
			noteExpected:       picoshare.FileNote{},
			expiresExpected:    mustParseExpirationTime("2024-12-15T21:52:33Z"),
			passphraseInStore:  "correct horse battery staple",
			passphraseExpected: "correct horse battery staple",
			status:             http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			originalData := "dummy original data"
			metadata := originalEntry
			metadata.Size = mustParseFileSize(len(originalData))
			if tt.passphraseInStore != "" {
				passphrase, err := picoshare.NewDownloadPassphrase(tt.passphraseInStore)
				if err != nil {
					t.Fatalf("failed to create download passphrase: %v", err)
				}
				metadata.DownloadPassphrase = passphrase
			}
			dataStore.InsertEntry(strings.NewReader((originalData)), metadata)
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			req := httptest.NewRequest(
				http.MethodPut,
				"/api/entry/"+tt.targetID,
				strings.NewReader(tt.payload),
			)
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

			if got, want := entry.DownloadPassphrase.String(), tt.passphraseExpected; got != want {
				t.Errorf("download passphrase=%q, want=%q", got, want)
			}
		})
	}
}

func TestGuestUpload(t *testing.T) {
	authenticator := shared_secret.New("dummypass")

	for _, tt := range []struct {
		description                string
		guestLinkInStore           picoshare.GuestLink
		entriesInStore             []picoshare.UploadEntry
		currentTime                time.Time
		url                        string
		note                       string
		downloadPassphrase         string
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
			description: "disabled guest link",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				IsDisabled:      true,
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
			description: "reject upload that includes a download passphrase",
			guestLinkInStore: picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			},
			currentTime:                mustParseTime("2024-01-01T00:00:00Z"),
			url:                        "/api/guest/abcdefgh23456789?expiration=2030-01-01T00:00:00Z",
			downloadPassphrase:         "correct horse battery staple",
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
			dataStore := test_sqlite.New(t)
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

			now := tt.currentTime
			s := handlers.New(authenticator, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })

			filename := "dummyimage.png"
			contents := "dummy bytes"
			formData, contentType := createMultipartFormBody(filename, tt.note, tt.downloadPassphrase, strings.NewReader(contents))

			req := httptest.NewRequest(http.MethodPost, tt.url, formData)
			req.Header.Add("Content-Type", contentType)
			req.Header.Add("Accept", "application/json")

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			entries, err := dataStore.GetEntriesMetadata()
			if err != nil {
				t.Fatalf("failed to list entries metadata: %v", err)
			}

			// On success, we expect the request to add a single entry to the store.
			// On failure, we expect no new entries in the store.
			expectedEntryCount := func() int {
				if tt.status == http.StatusOK {
					return len(tt.entriesInStore) + 1
				}
				return len(tt.entriesInStore)
			}()
			if got, want := len(entries), expectedEntryCount; got != want {
				t.Fatalf("entry count=%d, want=%d", got, want)
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
	authenticator := shared_secret.New("dummypass")

	for _, tt := range []struct {
		explanation         string
		acceptHeader        string
		expectJSON          bool
		expectedContentType string
	}{
		{
			explanation:         "no Accept header returns plain text URL",
			acceptHeader:        "",
			expectJSON:          false,
			expectedContentType: "text/plain",
		},
		{
			explanation:         "Accept header with wildcard returns plain text URL",
			acceptHeader:        "*/*",
			expectJSON:          false,
			expectedContentType: "text/plain",
		},
		{
			explanation:         "Accept header with application/json returns JSON",
			acceptHeader:        "application/json",
			expectJSON:          true,
			expectedContentType: "application/json",
		},
		{
			explanation:         "Accept header with text/html returns plain text URL",
			acceptHeader:        "text/html",
			expectJSON:          false,
			expectedContentType: "text/plain",
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.explanation, tt.acceptHeader), func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			guestLink := picoshare.GuestLink{
				ID:              picoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2022-05-26T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
			}
			if err := dataStore.InsertGuestLink(guestLink); err != nil {
				t.Fatalf("failed to insert dummy guest link: %v", err)
			}

			now := mustParseTime("2024-01-01T00:00:00Z")
			s := handlers.New(authenticator, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })

			filename := "dummyimage.png"
			contents := "dummy bytes"
			formData, contentType := createMultipartFormBody(filename, "", "", strings.NewReader(contents))

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/guest/abcdefgh23456789",
				formData,
			)
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

func createMultipartFormBody(filename, note, downloadPassphrase string, r io.Reader) (io.Reader, string) {
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

	pf, err := mw.CreateFormField("downloadPassphrase")
	if err != nil {
		panic(err)
	}
	pf.Write([]byte(downloadPassphrase))

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
