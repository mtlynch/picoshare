package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
)

func TestStaticResourceCaching(t *testing.T) {
	s := handlers.New(mockAuthenticator{}, nil, nilSpaceChecker, nilGarbageCollector, time.Now)

	var cssETag string
	{
		req := httptest.NewRequest(http.MethodGet, "/css/style.css", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Header().Get("Cache-Control"), expectedStaticCacheControl; got != want {
			t.Errorf("Cache-Control=%q, want=%q", got, want)
		}
		cssETag = rec.Header().Get("ETag")
		if got, want := cssETag != "", true; got != want {
			t.Fatalf("ETag present=%t, want=%t", got, want)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/css/style.css", nil)
		req.Header.Set("If-None-Match", cssETag)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusNotModified; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Body.Len(), 0; got != want {
			t.Errorf("body length=%d, want=%d", got, want)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/css/style.css", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Header().Get("ETag"), cssETag; got != want {
			t.Errorf("ETag=%q, want=%q", got, want)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/js/navbar.js", nil)
		rec := httptest.NewRecorder()

		s.Router().ServeHTTP(rec, req)

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status=%d, want=%d", got, want)
		}
		if got, want := rec.Header().Get("ETag") != cssETag, true; got != want {
			t.Errorf("ETag differs by path=%t, want=%t", got, want)
		}
	}
}

func TestStaticWebfontRangeRequest(t *testing.T) {
	s := handlers.New(mockAuthenticator{}, nil, nilSpaceChecker, nilGarbageCollector, time.Now)
	req := httptest.NewRequest(
		http.MethodGet,
		"/third-party/fontawesome6/webfonts/fa-solid-900.woff2",
		nil,
	)
	req.Header.Set("Range", "bytes=0-9")
	rec := httptest.NewRecorder()

	s.Router().ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusPartialContent; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
	if got, want := rec.Header().Get("Accept-Ranges"), "bytes"; got != want {
		t.Errorf("Accept-Ranges=%q, want=%q", got, want)
	}
	if got, want := strings.HasPrefix(rec.Header().Get("Content-Range"), "bytes 0-9/"), true; got != want {
		t.Errorf("Content-Range=%q, want prefix %q", rec.Header().Get("Content-Range"), "bytes 0-9/")
	}
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if got, want := len(body), 10; got != want {
		t.Errorf("body length=%d, want=%d", got, want)
	}
}
