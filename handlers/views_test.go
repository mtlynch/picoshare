package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
)

var errMultipleResponseWrites = errors.New("response body written more than once")

type unauthenticatedAuthenticator struct{}

func (unauthenticatedAuthenticator) StartSession(http.ResponseWriter, *http.Request) {}

func (unauthenticatedAuthenticator) ClearSession(http.ResponseWriter) {}

func (unauthenticatedAuthenticator) Authenticate(*http.Request) bool {
	return false
}

type singleWriteResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
	writeCalls int
}

func newSingleWriteResponseWriter() *singleWriteResponseWriter {
	return &singleWriteResponseWriter{
		header: make(http.Header),
	}
}

func (w *singleWriteResponseWriter) Header() http.Header {
	return w.header
}

func (w *singleWriteResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
}

func (w *singleWriteResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.writeCalls++
	if w.writeCalls > 1 {
		return 0, errMultipleResponseWrites
	}
	return w.body.Write(p)
}

func TestIndexGetWritesRenderedTemplateAtomically(t *testing.T) {
	s := handlers.New(
		unauthenticatedAuthenticator{},
		nil,
		nilSpaceCheck,
		nilGarbageCollector,
		time.Now,
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := newSingleWriteResponseWriter()

	s.Router().ServeHTTP(w, req)

	if got, want := w.statusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
	if got, want := strings.HasSuffix(strings.TrimSpace(w.body.String()), "</html>"), true; got != want {
		t.Errorf("response contains complete HTML document=%t, want=%t", got, want)
	}
}
