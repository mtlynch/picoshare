package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestFriendlyLinks(t *testing.T) {
	db := test_sqlite.New()
	clock := mockClock{t: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}
	s := handlers.New(mockAuthenticator{}, &db, nil, nil, clock)

	// 1. Upload a file with a friendly name
	filename := "test.txt"
	contents := "hello friendly world"
	friendlyName := "my-friendly-link"

	id1 := uploadWithFriendlyName(t, s, filename, contents, friendlyName, false)

	// 2. Retrieve the file via friendly name
	checkFriendlyLink(t, s, friendlyName, contents)

	// 3. Upload another file with the same friendly name, NO delete old
	contents2 := "updated content"
	id2 := uploadWithFriendlyName(t, s, filename, contents2, friendlyName, false)

	if id1 == id2 {
		t.Errorf("expected different IDs for different uploads, got %s", id1)
	}

	// Verify it now serves the new content
	checkFriendlyLink(t, s, friendlyName, contents2)

	// Verify old file still exists
	if _, err := db.GetEntryMetadata(id1); err != nil {
		t.Errorf("expected old file %s to still exist, but got error: %v", id1, err)
	}

	// 4. Upload another file with same friendly name, WITH delete old
	contents3 := "final content"
	_ = uploadWithFriendlyName(t, s, filename, contents3, friendlyName, true)

	// Verify it now serves the new content
	checkFriendlyLink(t, s, friendlyName, contents3)

	// Verify old file (id2) is deleted
	if _, err := db.GetEntryMetadata(id2); err == nil {
		t.Errorf("expected old file %s to be deleted, but it still exists", id2)
	}

	// Verify id1 still exists (because only the immediately previous one pointed to by the friendly link is deleted)
	if _, err := db.GetEntryMetadata(id1); err != nil {
		t.Errorf("expected file %s to still exist, but got error: %v", id1, err)
	}

	// 5. Verify disabled friendly link
	fl, err := db.GetFriendlyLink(picoshare.FriendlyName(friendlyName))
	if err != nil {
		t.Fatal(err)
	}
	fl.IsDisabled = true
	if err := db.UpdateFriendlyLink(fl); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/n/%s", friendlyName), nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusGone {
		t.Errorf("expected status %d for disabled friendly link, got %d", http.StatusGone, rr.Code)
	}

	// 6. Guest upload with friendly name
	guestLinkID := picoshare.GuestLinkID("guestinkidsixtee")
	gl := picoshare.GuestLink{
		ID:              guestLinkID,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		Created:         clock.t,
		UrlExpires:      picoshare.NeverExpire,
	}
	if err := db.InsertGuestLink(gl); err != nil {
		t.Fatal(err)
	}

	guestFriendlyName := "guest-friendly"
	guestContents := "guest content"
	_ = uploadGuestWithFriendlyName(t, s, "guest.txt", guestContents, guestLinkID, guestFriendlyName)

	checkFriendlyLink(t, s, guestFriendlyName, guestContents)
}

func TestFriendlyLinksCLI(t *testing.T) {
	db := test_sqlite.New()
	clock := mockClock{t: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}
	s := handlers.New(mockAuthenticator{}, &db, nil, nil, clock)

	guestLinkID := picoshare.GuestLinkID("guestinkidsixtee")
	gl := picoshare.GuestLink{
		ID:              guestLinkID,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		Created:         clock.t,
		UrlExpires:      picoshare.NeverExpire,
	}
	if err := db.InsertGuestLink(gl); err != nil {
		t.Fatal(err)
	}

	filename := "test.txt"
	contents := "hello cli world"
	friendlyName := "cli-friendly"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(contents)); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("friendly_name", friendlyName); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	// Simulating CLI (no Accept: application/json)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/guest/%s?expiration=2040-01-01T00:00:00Z", guestLinkID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("upload failed with status %d: %s", rr.Code, rr.Body.String())
	}

	expectedURL := fmt.Sprintf("http://example.com/n/%s\r\n", friendlyName)
	if rr.Body.String() != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, rr.Body.String())
	}
}

func TestFriendlyLinksQueryParams(t *testing.T) {
	db := test_sqlite.New()
	clock := mockClock{t: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}
	s := handlers.New(mockAuthenticator{}, &db, nil, nil, clock)

	guestLinkID := picoshare.GuestLinkID("guestinkidsixtee")
	gl := picoshare.GuestLink{
		ID:              guestLinkID,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		Created:         clock.t,
		UrlExpires:      picoshare.NeverExpire,
	}
	if err := db.InsertGuestLink(gl); err != nil {
		t.Fatal(err)
	}

	friendlyName := "query-friendly"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString("query params test")); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	// Parameters in query string
	url := fmt.Sprintf("/api/guest/%s?expiration=2040-01-01T00:00:00Z&friendly_name=%s&delete_old=true", guestLinkID, friendlyName)
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("upload failed with status %d: %s", rr.Code, rr.Body.String())
	}

	checkFriendlyLink(t, s, friendlyName, "query params test")
}

func uploadGuestWithFriendlyName(t *testing.T, s handlers.Server, filename, contents string, guestLinkID picoshare.GuestLinkID, friendlyName string) picoshare.EntryID {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(contents)); err != nil {
		t.Fatal(err)
	}
	if friendlyName != "" {
		if err := writer.WriteField("friendly_name", friendlyName); err != nil {
			t.Fatal(err)
		}
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/guest/%s?expiration=2040-01-01T00:00:00Z", guestLinkID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("guest upload failed with status %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	return picoshare.EntryID(resp.ID)
}

func uploadWithFriendlyName(t *testing.T, s handlers.Server, filename, contents, friendlyName string, deleteOld bool) picoshare.EntryID {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(contents)); err != nil {
		t.Fatal(err)
	}
	if friendlyName != "" {
		if err := writer.WriteField("friendly_name", friendlyName); err != nil {
			t.Fatal(err)
		}
	}
	if deleteOld {
		if err := writer.WriteField("delete_old", "true"); err != nil {
			t.Fatal(err)
		}
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/entry?expiration=2040-01-01T00:00:00Z", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("upload failed with status %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	return picoshare.EntryID(resp.ID)
}

func checkFriendlyLink(t *testing.T, s handlers.Server, friendlyName, expectedContents string) {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/n/%s", friendlyName), nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("failed to retrieve friendly link %s: status %d", friendlyName, rr.Code)
	}

	if rr.Body.String() != expectedContents {
		t.Errorf("expected contents %q, got %q", expectedContents, rr.Body.String())
	}
}
