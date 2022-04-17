package handlers_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
}

func TestUploadValidFile(t *testing.T) {
	store := test_sqlite.New()
	s := handlers.New(mockAuthenticator{}, store)

	filename := "dummyimage.png"
	contents := "dummy bytes"
	formData, contentType := createMultipartFormBody("file", filename, makeData(contents))

	req, err := http.NewRequest("POST", "/api/entry?expiration=2040-01-01T00:00:00Z", formData)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", contentType)

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response handlers.EntryPostResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("response is not valid JSON: %v", w.Body.String())
	}

	entry, err := store.GetEntry(types.EntryID(response.ID))
	if err != nil {
		t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
	}

	actual := mustReadAll(entry.Reader)
	expected := []byte(contents)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("stored entry doesn't match expected: got %v, want %v", actual, expected)
	}

	if entry.Filename != types.Filename(filename) {
		t.Fatalf("stored entry filename doesn't match expected: got %v, want %v", entry.Filename, filename)
	}

	expirationExpected := mustParseExpirationTime("2040-01-01T00:00:00Z")
	if entry.Expires != expirationExpected {
		t.Fatalf("stored entry expiration doesn't match expected: got %v, want %v", formatExpirationTime(entry.Expires), formatExpirationTime(expirationExpected))
	}
}

func TestEntryPostRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		description string
		name        string
		filename    string
		contents    string
	}{
		{
			description: "wrong form part name",
			name:        "badname",
			filename:    "dummy.png",
			contents:    "dummy bytes",
		},
		{
			description: "filename that's just a dot",
			name:        "file",
			filename:    ".",
			contents:    "dummy bytes",
		},
		{
			description: "empty upload",
			name:        "file",
			filename:    "dummy.png",
			contents:    "",
		},
	}
	for _, tt := range tests {
		store := test_sqlite.New()
		s := handlers.New(mockAuthenticator{}, store)

		formData, contentType := createMultipartFormBody(tt.name, tt.filename, bytes.NewBuffer([]byte(tt.contents)))

		req, err := http.NewRequest("POST", "/api/entry", formData)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Content-Type", contentType)

		w := httptest.NewRecorder()
		s.Router().ServeHTTP(w, req)

		if status := w.Code; status != http.StatusBadRequest {
			t.Errorf("%s: handler returned wrong status code: got %v want %v",
				tt.description, status, http.StatusBadRequest)
		}
	}
}

func TestGuestUploadValidFile(t *testing.T) {
	store := test_sqlite.New()
	store.InsertGuestLink(types.GuestLink{
		ID:      types.GuestLinkID("abcdefgh23456789"),
		Created: mustParseTime("2022-01-01T00:00:00Z"),
		Expires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})

	authenticator, err := shared_secret.New("dummypass")
	if err != nil {
		t.Fatalf("failed to create shared secret: %v", err)
	}

	s := handlers.New(authenticator, store)

	filename := "dummyimage.png"
	contents := "dummy bytes"
	formData, contentType := createMultipartFormBody("file", filename, makeData(contents))

	req, err := http.NewRequest("POST", "/api/guest/abcdefgh23456789", formData)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", contentType)

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response handlers.EntryPostResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("response is not valid JSON: %v", w.Body.String())
	}

	entry, err := store.GetEntry(types.EntryID(response.ID))
	if err != nil {
		t.Fatalf("failed to get expected entry %v from data store: %v", response.ID, err)
	}

	actual := mustReadAll(entry.Reader)
	expected := []byte(contents)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("stored entry doesn't match expected: got %v, want %v", actual, expected)
	}

	if entry.Filename != types.Filename(filename) {
		t.Fatalf("stored entry filename doesn't match expected: got %v, want %v", entry.Filename, filename)
	}

	if entry.Expires != types.NeverExpire {
		t.Fatalf("stored entry expiration doesn't match expected: got %v, want %v", formatExpirationTime(entry.Expires), formatExpirationTime(types.NeverExpire))
	}
}

func TestGuestUploadInvalidLink(t *testing.T) {
	tests := []struct {
		description      string
		guestLinkInStore types.GuestLink
		entriesInStore   []types.UploadEntry
		guestLinkID      string
		statusExpected   int
	}{
		{
			description: "expired guest link",
			guestLinkInStore: types.GuestLink{
				ID:      types.GuestLinkID("abcdefgh23456789"),
				Created: mustParseTime("2000-01-01T00:00:00Z"),
				Expires: mustParseExpirationTime("2000-01-02T03:04:25Z"),
			},
			guestLinkID:    "abcdefgh23456789",
			statusExpected: http.StatusUnauthorized,
		},
		{
			description: "invalid guest link",
			guestLinkInStore: types.GuestLink{
				ID:      types.GuestLinkID("abcdefgh23456789"),
				Created: mustParseTime("2000-01-01T00:00:00Z"),
				Expires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
			},
			guestLinkID:    "i-am-an-invalid-guest-link",
			statusExpected: http.StatusBadRequest,
		},
		{
			description: "non-existent guest link",
			guestLinkInStore: types.GuestLink{
				ID:      types.GuestLinkID("abcdefgh23456789"),
				Created: mustParseTime("2000-01-01T00:00:00Z"),
				Expires: mustParseExpirationTime("2000-01-02T03:04:25Z"),
			},
			guestLinkID:    "doesntexistaaaaa",
			statusExpected: http.StatusNotFound,
		},
		{
			description: "exhausted upload count",
			guestLinkInStore: types.GuestLink{
				ID:             types.GuestLinkID("abcdefgh23456789"),
				Created:        mustParseTime("2000-01-01T00:00:00Z"),
				Expires:        mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileUploads: makeGuestUploadCountLimit(2),
			},
			entriesInStore: []types.UploadEntry{
				{
					UploadMetadata: types.UploadMetadata{
						ID:          types.EntryID("dummy-entry1"),
						GuestLinkID: types.GuestLinkID("abcdefgh23456789"),
					},
				},
				{
					UploadMetadata: types.UploadMetadata{
						ID:          types.EntryID("dummy-entry2"),
						GuestLinkID: types.GuestLinkID("abcdefgh23456789"),
					},
				},
			},
			guestLinkID:    "abcdefgh23456789",
			statusExpected: http.StatusUnauthorized,
		},
		{
			description: "exhausted upload count",
			guestLinkInStore: types.GuestLink{
				ID:           types.GuestLinkID("abcdefgh23456789"),
				Created:      mustParseTime("2000-01-01T00:00:00Z"),
				Expires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileBytes: makeGuestUploadMaxFileBytes(1),
			},
			guestLinkID:    "abcdefgh23456789",
			statusExpected: http.StatusBadRequest,
		},
	}

	authenticator, err := shared_secret.New("dummypass")
	if err != nil {
		t.Fatalf("failed to create shared secret: %v", err)
	}

	for _, tt := range tests {
		store := test_sqlite.New()
		if err := store.InsertGuestLink(tt.guestLinkInStore); err != nil {
			t.Fatalf("%s: failed to insert dummy guest link: %v", tt.description, err)
		}
		for _, entry := range tt.entriesInStore {
			if err := store.InsertEntry(strings.NewReader("dummy data"), entry.UploadMetadata); err != nil {
				t.Fatalf("%s: failed to insert dummy entry: %v", tt.description, err)
			}
		}

		s := handlers.New(authenticator, store)

		filename := "dummyimage.png"
		contents := "dummy bytes"
		formData, contentType := createMultipartFormBody("file", filename, makeData(contents))

		req, err := http.NewRequest("POST", "/api/guest/"+tt.guestLinkID, formData)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Content-Type", contentType)

		w := httptest.NewRecorder()
		s.Router().ServeHTTP(w, req)

		if status := w.Code; status != tt.statusExpected {
			t.Fatalf("%s: handler returned wrong status code: got %v want %v",
				tt.description, status, tt.statusExpected)
		}
	}
}

func createMultipartFormBody(name, filename string, r io.Reader) (io.Reader, string) {
	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)

	part, err := mw.CreateFormFile(name, filename)
	if err != nil {
		panic(err)
	}
	io.Copy(part, r)

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

func mustParseExpirationTime(s string) types.ExpirationTime {
	return types.ExpirationTime(mustParseTime(s))
}

func formatExpirationTime(et types.ExpirationTime) string {
	return time.Time(et).Format(time.RFC3339)
}

func mustReadAll(r io.Reader) []byte {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return d
}
