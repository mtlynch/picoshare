package handlers_test

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

type mockAuthenticator struct{}

func (ma mockAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ma mockAuthenticator) ClearSession(w http.ResponseWriter) {}

func (ma mockAuthenticator) Authenticate(r *http.Request) bool {
	return true
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
