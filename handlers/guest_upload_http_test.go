package handlers_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestParallelGuestUploadRespondsBeforeReceivingHTTPBody(t *testing.T) {
	dataStore := test_sqlite.New(t)
	now := mustParseTime("2025-01-01T00:00:00Z")
	mustInsertGuestUploadLink(t, &dataStore, picoshare.GuestLink{
		ID:              "abcdefgh23456789",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(2),
	})
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, func() time.Time { return now })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	release := make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(release) })
	firstDone := make(chan struct{})
	defer func() { releaseFirst(); <-firstDone }()
	firstStarted := make(chan struct{})
	first := httptest.NewRecorder()
	{
		body, contentType := createMultipartFormBody("first.txt", "", "", strings.NewReader("first payload"))
		heldBody := &beforeGuestUploadRead{Reader: body, beforeRead: func() {
			close(firstStarted)
			<-release
		}}
		req := httptest.NewRequest(http.MethodPost, "/api/guest/abcdefgh23456789", heldBody)
		req.Header.Set("Content-Type", contentType)
		go func() {
			s.Router().ServeHTTP(first, req)
			close(firstDone)
		}()
	}
	select {
	case <-firstStarted:
	case <-firstDone:
		t.Fatalf("first upload finished before reading its body: status=%d", first.Code)
	case <-ctx.Done():
		t.Fatalf("first upload did not start: %v", ctx.Err())
	}

	// Use a real HTTP/1.1 connection: ResponseRecorder does not model the
	// server draining unread request bodies before writing its response.
	server := httptest.NewServer(s.Router())
	defer server.Close()
	{
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", server.Listener.Addr().String())
		if err != nil {
			t.Fatalf("failed to connect to HTTP server: %v", err)
		}
		// Close the client before server.Close, including on assertion failures.
		defer conn.Close()
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("test context has no deadline")
		}
		if err := conn.SetDeadline(deadline); err != nil {
			t.Fatalf("failed to set connection deadline: %v", err)
		}
		// Declare a small body without sending it or Expect: 100-continue, which
		// would hide the server's automatic body-draining behavior.
		requestHeaders := "POST /api/guest/abcdefgh23456789 HTTP/1.1\r\n" +
			"Host: example.com\r\n" +
			"Content-Type: multipart/form-data; boundary=held\r\n" +
			"Content-Length: 100\r\n\r\n"
		if _, err := io.WriteString(conn, requestHeaders); err != nil {
			t.Fatalf("failed to send request headers: %v", err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatalf("failed to read rejection before sending request body: %v", err)
		}
		defer response.Body.Close()
		if got, want := response.StatusCode, http.StatusConflict; got != want {
			t.Fatalf("parallel upload status=%d, want=%d", got, want)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("failed to read rejection body: %v", err)
		}
		if got, want := string(body), "Parallel uploads are forbidden on this guest link\n"; got != want {
			t.Errorf("rejection=%q, want=%q", got, want)
		}
	}
	releaseFirst()
	<-firstDone
	if got, want := first.Code, http.StatusOK; got != want {
		t.Fatalf("first upload status=%d, want=%d: %s", got, want, first.Body.String())
	}
}
