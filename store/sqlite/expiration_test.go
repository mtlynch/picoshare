package sqlite_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
	"github.com/mtlynch/picoshare/testutil"
)

func TestReadEntryWithEpochExpiration(t *testing.T) {
	db := test_sqlite.New(t)
	epoch := time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC)
	expiration := testutil.MustCreate(t, picoshare.NewExpirationTime, epoch)
	entryID := picoshare.MustCreateEntryID("abcdefghij")
	if err := db.InsertEntry(strings.NewReader("hello"), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "expired.txt",
		Uploaded: epoch,
		Expires:  expiration,
	}); err != nil {
		t.Fatalf("failed to insert expired entry: %v", err)
	}

	metadata, err := db.GetEntryMetadata(entryID)
	if err != nil {
		t.Fatalf("failed to read expired entry: %v", err)
	}
	if got, want := metadata.Expires, expiration; got != want {
		t.Errorf("expiration=%v, want=%v", got, want)
	}

	entries, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to list expired entries: %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("entry count=%d, want=%d", got, want)
	}
	if got, want := entries[0].Expires, expiration; got != want {
		t.Errorf("listed expiration=%v, want=%v", got, want)
	}
}

func TestReadGuestLinkWithEpochExpiration(t *testing.T) {
	db := test_sqlite.New(t)
	epoch := time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC)
	expiration := testutil.MustCreate(t, picoshare.NewExpirationTime, epoch)
	guestLinkID := picoshare.GuestLinkID("guest-link")
	if err := db.InsertGuestLink(picoshare.GuestLink{
		ID:              guestLinkID,
		Created:         epoch,
		UrlExpires:      expiration,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	}); err != nil {
		t.Fatalf("failed to insert expired guest link: %v", err)
	}

	guestLink, err := db.GetGuestLink(guestLinkID)
	if err != nil {
		t.Fatalf("failed to read expired guest link: %v", err)
	}
	if got, want := guestLink.UrlExpires, expiration; got != want {
		t.Errorf("expiration=%v, want=%v", got, want)
	}
	if got, want := guestLink.IsActive(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)), false; got != want {
		t.Errorf("guest link active=%v, want=%v", got, want)
	}
}
