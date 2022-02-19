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

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store/memory"
	"github.com/mtlynch/picoshare/v2/types"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
}

func TestUploadValidFile(t *testing.T) {
	store := memory.New()
	s := handlers.New(mockAuthenticator{}, store)

	filename := "dummyimage.png"
	contents := []byte("dummy bytes")
	formData, contentType := createMultipartFormBody("file", filename, contents)

	req, err := http.NewRequest("POST", "/api/entry", formData)
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

	if !reflect.DeepEqual(entry.Data, contents) {
		t.Fatalf("stored entry doesn't match expected: got %v, want %v", entry.Data, contents)
	}

	if entry.Filename != types.Filename(filename) {
		t.Fatalf("stored entry filename doesn't match expected: got %v, want %v", entry.Filename, filename)
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
			description: "filename with backslashes",
			name:        "file",
			filename:    `filename\with\backslashes.png`,
			contents:    "dummy bytes",
		},
		{
			description: "filename that's just a dot",
			name:        "file",
			filename:    ".",
			contents:    "dummy bytes",
		},
		{
			description: "filename that's two dots",
			name:        "file",
			filename:    "..",
			contents:    "dummy bytes",
		},
		{
			description: "filename that's five dots",
			name:        "file",
			filename:    ".....",
			contents:    "dummy bytes",
		},
		{
			description: "filename that's too long",
			name:        "file",
			filename:    strings.Repeat("A", handlers.MaxFilenameLen+1),
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
		store := memory.New()
		s := handlers.New(mockAuthenticator{}, store)

		formData, contentType := createMultipartFormBody(tt.name, tt.filename, []byte(tt.contents))

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

func createMultipartFormBody(name, filename string, contents []byte) (io.Reader, string) {
	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)

	part, err := mw.CreateFormFile(name, filename)
	if err != nil {
		panic(err)
	}
	part.Write(contents)

	mw.Close()
	bw.Flush()

	return bufio.NewReader(&b), mw.FormDataContentType()
}
