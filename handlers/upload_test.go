package handlers_test

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
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
