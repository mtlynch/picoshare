package handlers_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
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

	formData, contentType := createMultipartFormBody("file", "dummyimage.png", "dummy bytes")
	d, _ := ioutil.ReadAll(formData)
	log.Print(string(d))

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
}

func createMultipartFormBody(name, filename, contents string) (io.Reader, string) {
	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	mw := multipart.NewWriter(bw)
	defer mw.Close()

	part, err := mw.CreateFormFile(name, filename)
	if err != nil {
		panic(err)
	}
	part.Write([]byte(contents))

	bw.Flush()

	return bufio.NewReader(&b), mw.FormDataContentType()
}
