package handlers_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type heldUploadBody struct {
	started chan struct{}
	release chan struct{}
}

func (b heldUploadBody) Read([]byte) (int, error) {
	close(b.started)
	<-b.release
	return 0, io.ErrUnexpectedEOF
}

func (b heldUploadBody) Close() error { return nil }

type observedUploadBody struct {
	read bool
}

func (b *observedUploadBody) Read([]byte) (int, error) {
	b.read = true
	return 0, io.EOF
}

func (b *observedUploadBody) Close() error { return nil }

func TestGuestReservationPrecedesBodyRead(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	limit := 1
	link := picoshare.GuestLink{
		ID: "abcdefgh23456789", Created: now,
		UrlExpires: picoshare.NeverExpire, MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads: &limit,
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	body := heldUploadBody{started: make(chan struct{}), release: make(chan struct{})}
	firstDone := make(chan struct{})
	firstResponse := httptest.NewRecorder()
	{
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
		go func() {
			defer close(firstDone)
			s.Router().ServeHTTP(firstResponse, req)
		}()
	}
	select {
	case <-body.started:
	case <-firstDone:
		t.Fatalf("first request ended before reading its body: %d %s", firstResponse.Code, firstResponse.Body.String())
	}
	// Always unblock the first request, including when an assertion fails.
	defer func() { close(body.release); <-firstDone }()
	{
		secondBody := &observedUploadBody{}
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", secondBody)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("status=%d, want=%d: %s", got, want, rec.Body.String())
		}
		if got, want := secondBody.read, false; got != want {
			t.Errorf("rejected request body read=%v, want=%v", got, want)
		}
	}
}

func TestGuestMultipartFailureReleasesReservation(t *testing.T) {
	for _, tt := range []struct {
		explanation        string
		duplicateFile      bool
		missingFile        bool
		truncateBody       bool
		trailingNote       string
		trailingPassphrase string
		query              string
	}{
		{
			explanation:   "duplicate file parts should reject the upload",
			duplicateFile: true,
		},
		{
			explanation: "missing file should reject the upload",
			missingFile: true,
		},
		{
			explanation:  "truncated multipart should reject the upload",
			truncateBody: true,
		},
		{
			explanation:  "a note after the file should reject the upload",
			trailingNote: "forbidden",
		},
		{
			explanation:        "a passphrase after the file should reject the upload",
			trailingPassphrase: "forbidden",
		},
		{
			explanation: "a note in the query should reject the upload",
			query:       "?note=forbidden",
		},
		{
			explanation: "a passphrase in the query should reject the upload",
			query:       "?downloadPassphrase=forbidden",
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.NewWithChunkSize(t, 2)
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			limit := 1
			link := picoshare.GuestLink{
				ID: "abcdefgh23456789", Created: now,
				UrlExpires: picoshare.NeverExpire, MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileUploads: &limit,
			}
			if err := dataStore.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
			{
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				if !tt.missingFile {
					part, err := writer.CreateFormFile("file", "first.txt")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := io.WriteString(part, "first file contents"); err != nil {
						t.Fatal(err)
					}
				}
				if tt.duplicateFile {
					part, err := writer.CreateFormFile("file", "second.txt")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := io.WriteString(part, "second file contents"); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.WriteField("note", tt.trailingNote); err != nil {
					t.Fatal(err)
				}
				if err := writer.WriteField("downloadPassphrase", tt.trailingPassphrase); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				payload := body.Bytes()
				if tt.truncateBody {
					payload = payload[:len(payload)-10]
				}
				req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789"+tt.query, bytes.NewReader(payload))
				req.Header.Set("Content-Type", writer.FormDataContentType())
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				if got, want := rec.Code, http.StatusBadRequest; got != want {
					t.Fatalf("status=%d, want=%d: %s", got, want, rec.Body.String())
				}
				entries, err := dataStore.GetEntriesMetadata()
				if err != nil {
					t.Fatal(err)
				}
				if got, want := len(entries), 0; got != want {
					t.Fatalf("entries=%d, want=%d", got, want)
				}
			}
			{
				body, contentType := createMultipartFormBody("retry.txt", "", "", strings.NewReader("retry contents"))
				req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", body)
				req.Header.Set("Content-Type", contentType)
				rec := httptest.NewRecorder()
				s.Router().ServeHTTP(rec, req)
				if got, want := rec.Code, http.StatusOK; got != want {
					t.Fatalf("retry status=%d, want=%d: %s", got, want, rec.Body.String())
				}
			}
		})
	}
}
