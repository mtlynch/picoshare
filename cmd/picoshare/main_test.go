package main_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	gorilla "github.com/mtlynch/gorilla-handlers"
	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

func TestUpload(t *testing.T) {
	const fileSize = 5 * (1 << 30) // 5GB

	// Start HTTP server and wait a moment for it to kick in.
	store := sqlite.New(filepath.Join(t.TempDir(), "db"))
	http.Handle("/", gorilla.LoggingHandler(os.Stdout, handlers.New(nil, store).Router()))
	go http.ListenAndServe(":9000", nil)
	time.Sleep(1 * time.Second)

	// Write the multi-part form through a pipe so we don't need to allocate it ahead of time.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, err := mw.CreateFormFile("file", "foo.dat")
		if err != nil {
			t.Fatal(err)
		} else if _, err := io.CopyN(fw, &zeroReader{}, fileSize); err != nil {
			t.Fatal(err)
		}

		if err := mw.Close(); err != nil {
			t.Fatal(err)
		} else if err := pw.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Send request and hope for the best.
	req, err := http.NewRequest("POST", "http://localhost:9000/api/entry?expiration=2030-01-01T00:00:00Z", pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, 200; got != want {
		t.Fatalf("StatusCode=%v, want %v", got, want)
	}
}

// zeroReader implements io.Reader and always fills the buffer with zeros.
type zeroReader struct{}

func (r *zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
