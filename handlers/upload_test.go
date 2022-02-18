package handlers_test

import (
	"bufio"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store/memory"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
}

func TestUploadFile(t *testing.T) {
	store := memory.New()
	s := handlers.New(mockAuthenticator{}, store)

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)
	part, err := mw.CreateFormFile("file", "someimg.png")
	if err != nil {
		t.Error(err)
	}
	part.Write([]byte("dummy bytes"))

	mw.Close()
	bw.Flush()

	req, err := http.NewRequest("POST", "/api/entry", bufio.NewReader(&b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
