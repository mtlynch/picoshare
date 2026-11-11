package sqlite_test

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/random"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/sqlite"
)

type guestUploadReadFunc func([]byte) (int, error)

func (f guestUploadReadFunc) Read(p []byte) (int, error) {
	return f(p)
}

func TestInsertEntryRejectsGuestUploadWhenCountLimitDropsToExistingCount(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	// Use one of two upload slots so the next upload is initially allowed.
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(2),
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	priorMetadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("priorentry"), link, now)
	if err := dataStore.InsertEntry(strings.NewReader("prior"), priorMetadata); err != nil {
		t.Fatalf("failed to insert prior upload: %v", err)
	}
	// The upload retains the original link snapshot, not its current policy.
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)
	metadata.Size = mustParseFileSize(8)
	policyApplied := false
	reader := io.MultiReader(strings.NewReader("abcdefgh"), guestUploadReadFunc(func([]byte) (int, error) {
		// Both full chunks have been persisted before the policy changes.
		assertGuestUploadRows(t, db, metadata.ID, 0, 2)
		// Remove the remaining slot before InsertEntry commits the upload.
		if _, err := db.Exec(`UPDATE guest_links SET max_file_uploads = 1 WHERE id = ?`, link.ID); err != nil {
			t.Fatalf("failed to reduce guest link upload limit during payload read: %v", err)
		}
		policyApplied = true
		return 0, io.EOF
	}))

	err := dataStore.InsertEntry(reader, metadata)
	if got, want := policyApplied, true; got != want {
		t.Fatalf("final policy applied=%v, want=%v", got, want)
	}
	// The current limit must reject the upload and leave no staged data behind.
	if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
		t.Fatalf("policy rejection=%v, want=%v (err=%v)", got, want, err)
	}
	assertGuestUploadRows(t, db, metadata.ID, 0, 0)
}

func TestInsertEntryAcceptsGuestUploadInLastAvailableSlot(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	// A new link with a limit of one has exactly one available upload slot.
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)

	if err := dataStore.InsertEntry(strings.NewReader("abcdefgh"), metadata); err != nil {
		t.Fatalf("failed to insert upload in last available slot: %v", err)
	}
	// Filling the last slot must persist the file and count it as one upload.
	assertGuestUploadRows(t, db, metadata.ID, 1, 2)
	stored, err := dataStore.GetEntryMetadata(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read accepted upload metadata: %v", err)
	}
	if got, want := stored.GuestLink.FilesUploaded, 1; got != want {
		t.Errorf("link upload count=%d, want=%d", got, want)
	}
}

func TestInsertEntryExcludesOtherGuestLinksUploadsFromCountLimit(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	// Exhaust a separate link's capacity while leaving this link unused.
	otherLink := link
	otherLink.ID = "other-link"
	if err := dataStore.InsertGuestLink(otherLink); err != nil {
		t.Fatalf("failed to insert other guest link: %v", err)
	}
	otherMetadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("otherentry"), otherLink, now)
	if err := dataStore.InsertEntry(strings.NewReader("other"), otherMetadata); err != nil {
		t.Fatalf("failed to insert upload for other link: %v", err)
	}
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)

	// The other link's upload must not prevent this link from accepting one.
	if err := dataStore.InsertEntry(strings.NewReader("abcdefgh"), metadata); err != nil {
		t.Fatalf("failed to insert upload with capacity consumed only on another link: %v", err)
	}
	// This link's count must include only its own newly accepted upload.
	assertGuestUploadRows(t, db, metadata.ID, 1, 2)
	stored, err := dataStore.GetEntryMetadata(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read accepted upload metadata: %v", err)
	}
	if got, want := stored.GuestLink.FilesUploaded, 1; got != want {
		t.Errorf("link upload count=%d, want=%d", got, want)
	}
}

func TestInsertEntryAcceptsAdditionalGuestUploadWithoutCountLimit(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	// Leave MaxFileUploads unset so the link permits unlimited uploads.
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	// Give the link an existing upload before attempting another.
	priorMetadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("priorentry"), link, now)
	if err := dataStore.InsertEntry(strings.NewReader("prior"), priorMetadata); err != nil {
		t.Fatalf("failed to insert prior upload: %v", err)
	}
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)

	if err := dataStore.InsertEntry(strings.NewReader("abcdefgh"), metadata); err != nil {
		t.Fatalf("failed to insert additional upload without count limit: %v", err)
	}
	// Unlimited links still count uploads even though the count cannot reject one.
	assertGuestUploadRows(t, db, metadata.ID, 1, 2)
	stored, err := dataStore.GetEntryMetadata(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read accepted upload metadata: %v", err)
	}
	if got, want := stored.GuestLink.FilesUploaded, 2; got != want {
		t.Errorf("link upload count=%d, want=%d", got, want)
	}
}

func TestInsertEntryEnforcesGuestUploadByteLimitUsingActualSize(t *testing.T) {
	for _, tt := range []struct {
		explanation  string
		maxFileBytes picoshare.GuestUploadMaxFileBytes
		reportedSize int
		wantRejected bool
	}{
		{
			explanation:  "unlimited file size accepts a payload larger than its reported size",
			reportedSize: 1,
		},
		{
			explanation:  "a payload exactly at the byte limit succeeds",
			maxFileBytes: new(uint64(8)),
			reportedSize: 8,
		},
		{
			explanation:  "underreported metadata cannot bypass the actual byte limit",
			maxFileBytes: new(uint64(7)),
			reportedSize: 1,
			wantRejected: true,
		},
		{
			explanation:  "overreported metadata does not reject a payload within the byte limit",
			maxFileBytes: new(uint64(8)),
			reportedSize: 100,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := mustParseTime("2035-01-01T00:00:00Z")
			dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
			link := picoshare.GuestLink{
				ID:              "guest-link",
				Created:         now,
				UrlExpires:      picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite,
				MaxFileBytes:    tt.maxFileBytes,
			}
			if err := dataStore.InsertGuestLink(link); err != nil {
				t.Fatalf("failed to insert guest link: %v", err)
			}
			// Keep the payload at eight bytes while varying the reported size
			// and byte limit independently.
			metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)
			metadata.Size = mustParseFileSize(tt.reportedSize)
			payload := "abcdefgh"

			err := dataStore.InsertEntry(strings.NewReader(payload), metadata)
			if got, want := err != nil, tt.wantRejected; got != want {
				t.Fatalf("upload rejected=%v, want=%v (err=%v)", got, want, err)
			}
			if tt.wantRejected {
				// An oversized payload must leave neither metadata nor chunks.
				if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
					t.Fatalf("policy rejection=%v, want=%v (err=%v)", got, want, err)
				}
				assertGuestUploadRows(t, db, metadata.ID, 0, 0)
				return
			}
			// Accepted uploads must store their actual size, not the supplied size.
			assertGuestUploadRows(t, db, metadata.ID, 1, 2)
			stored, err := dataStore.GetEntryMetadata(metadata.ID)
			if err != nil {
				t.Fatalf("failed to read accepted upload metadata: %v", err)
			}
			if got, want := stored.Size, mustParseFileSize(len(payload)); !got.Equal(want) {
				t.Errorf("stored size=%v, want=%v", got, want)
			}
			if got, want := stored.GuestLink.FilesUploaded, 1; got != want {
				t.Errorf("link upload count=%d, want=%d", got, want)
			}
		})
	}
}

func TestInsertEntryRejectsGuestUploadWhenByteLimitDropsBelowPayloadSize(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	// Initially allow the entire eight-byte payload.
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileBytes:    new(uint64(8)),
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	// The upload retains the original link snapshot, not its current policy.
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)
	metadata.Size = mustParseFileSize(8)
	policyApplied := false
	reader := io.MultiReader(strings.NewReader("abcdefgh"), guestUploadReadFunc(func([]byte) (int, error) {
		// Both full chunks have been persisted before the policy changes.
		assertGuestUploadRows(t, db, metadata.ID, 0, 2)
		// Make the staged payload too large before InsertEntry commits it.
		if _, err := db.Exec(`UPDATE guest_links SET max_file_bytes = 7 WHERE id = ?`, link.ID); err != nil {
			t.Fatalf("failed to reduce guest link byte limit during payload read: %v", err)
		}
		policyApplied = true
		return 0, io.EOF
	}))

	err := dataStore.InsertEntry(reader, metadata)
	if got, want := policyApplied, true; got != want {
		t.Fatalf("final policy applied=%v, want=%v", got, want)
	}
	// Recheck the new limit despite the permissive snapshot and remove the chunks.
	if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
		t.Fatalf("policy rejection=%v, want=%v (err=%v)", got, want, err)
	}
	assertGuestUploadRows(t, db, metadata.ID, 0, 0)
}

func TestInsertEntryGuestUploadsCompeteForLastSlot(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	firstStore, db, path := openGuestUploadStore(t, func() time.Time { return now })
	secondStore := sqlite.New(sqlite.Params{
		Path:               path,
		Now:                func() time.Time { return now },
		PicoShareChunkSize: 4,
	})
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads:  new(1),
	}
	if err := firstStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	firstMetadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("firstentry"), link, now)
	secondMetadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("secondentr"), link, now)
	firstStaged := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstResult := make(chan error, 1)
	go func() {
		reader := io.MultiReader(strings.NewReader("abcdefgh"), guestUploadReadFunc(func([]byte) (int, error) {
			close(firstStaged)
			<-releaseFirst
			return 0, io.EOF
		}))
		firstResult <- firstStore.InsertEntry(reader, firstMetadata)
	}()
	<-firstStaged
	// Release and join the blocked upload even if the competing upload fails.
	firstJoined := false
	t.Cleanup(func() {
		if !firstJoined {
			close(releaseFirst)
			<-firstResult
		}
	})
	assertGuestUploadRows(t, db, firstMetadata.ID, 0, 2)

	secondReader := io.MultiReader(strings.NewReader("ijklmnop"), guestUploadReadFunc(func([]byte) (int, error) {
		// Two independent stores have staged data before either commits metadata.
		assertGuestUploadRows(t, db, firstMetadata.ID, 0, 2)
		assertGuestUploadRows(t, db, secondMetadata.ID, 0, 2)
		return 0, io.EOF
	}))
	if err := secondStore.InsertEntry(secondReader, secondMetadata); err != nil {
		t.Fatalf("failed to claim the last slot from the second store: %v", err)
	}
	close(releaseFirst)
	err := <-firstResult
	firstJoined = true
	if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
		t.Errorf("first upload policy rejection=%v, want=%v (err=%v)", got, want, err)
	}
	assertGuestUploadRows(t, db, firstMetadata.ID, 0, 0)
	assertGuestUploadRows(t, db, secondMetadata.ID, 1, 2)
	storedLink, err := firstStore.GetGuestLink(link.ID)
	if err != nil {
		t.Fatalf("failed to read guest link: %v", err)
	}
	if got, want := storedLink.FilesUploaded, 1; got != want {
		t.Errorf("link upload count=%d, want=%d", got, want)
	}
}

func TestInsertEntryRejectedGuestUploadCleanupFailure(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
		IsDisabled:      true,
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert disabled guest link: %v", err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_chunk_cleanup BEFORE DELETE ON entries_data BEGIN SELECT RAISE(ABORT, 'chunk cleanup failed'); END`); err != nil {
		t.Fatalf("failed to create cleanup failure trigger: %v", err)
	}
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)

	err := dataStore.InsertEntry(strings.NewReader("abcdefgh"), metadata)
	if err == nil {
		t.Fatal("upload unexpectedly succeeded despite policy rejection and failed cleanup")
	}
	if got, want := errors.Is(err, store.ErrGuestUploadRejected), false; got != want {
		t.Errorf("policy rejection despite failed cleanup=%v, want=%v (err=%v)", got, want, err)
	}
	if got, want := err.Error(), "failed to remove chunks"; !strings.Contains(got, want) {
		t.Errorf("error=%q, want containing %q", got, want)
	}
	if got, want := err.Error(), "chunk cleanup failed"; !strings.Contains(got, want) {
		t.Errorf("error=%q, want containing %q", got, want)
	}
	assertGuestUploadRows(t, db, metadata.ID, 0, 2)
}

func TestInsertEntryGuestRejectionPreservesExistingEntry(t *testing.T) {
	now := mustParseTime("2035-01-01T00:00:00Z")
	dataStore, db, _ := openGuestUploadStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{
		ID:              "guest-link",
		Created:         now,
		UrlExpires:      picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite,
	}
	if err := dataStore.InsertGuestLink(link); err != nil {
		t.Fatalf("failed to insert guest link: %v", err)
	}
	metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("existingab"), link, now)
	metadata.Filename = "original.txt"
	contents := "original"
	if err := dataStore.InsertEntry(strings.NewReader(contents), metadata); err != nil {
		t.Fatalf("failed to insert original upload: %v", err)
	}
	if err := dataStore.DisableGuestLink(link.ID); err != nil {
		t.Fatalf("failed to disable guest link: %v", err)
	}
	originalMetadata, err := dataStore.GetEntryMetadata(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read original metadata: %v", err)
	}
	retryMetadata := metadata
	retryMetadata.Filename = "replacement.txt"
	retryMetadata.Expires = mustParseExpirationTime("2040-01-01T00:00:00Z")

	// An empty payload reaches policy rejection without a chunk-key collision.
	err = dataStore.InsertEntry(strings.NewReader(""), retryMetadata)
	if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
		t.Errorf("policy rejection=%v, want=%v (err=%v)", got, want, err)
	}
	assertGuestUploadRows(t, db, metadata.ID, 1, 2)
	storedMetadata, err := dataStore.GetEntryMetadata(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read preserved metadata: %v", err)
	}
	if got, want := storedMetadata, originalMetadata; got != want {
		t.Errorf("preserved metadata=%+v, want=%+v", got, want)
	}
	reader, err := dataStore.ReadEntryFile(metadata.ID)
	if err != nil {
		t.Fatalf("failed to read preserved file: %v", err)
	}
	storedContents, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read preserved contents: %v", err)
	}
	if got, want := string(storedContents), contents; got != want {
		t.Errorf("preserved contents=%q, want=%q", got, want)
	}
}

func openGuestUploadStore(t *testing.T, now func() time.Time) (sqlite.Store, *sql.DB, string) {
	t.Helper()
	name := random.String(20, []rune("abcdefghijklmnopqrstuvwxyz"))
	path := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	dataStore := sqlite.New(sqlite.Params{
		Path:               path,
		Now:                now,
		PicoShareChunkSize: 4,
	})
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("failed to open shared in-memory database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close shared in-memory database: %v", err)
		}
	})
	return dataStore, db, path
}

func createGuestUploadMetadata(id picoshare.EntryID, link picoshare.GuestLink, now time.Time) picoshare.UploadMetadata {
	return picoshare.UploadMetadata{
		ID:        id,
		Filename:  "guest-upload.txt",
		GuestLink: link,
		Uploaded:  now,
		Expires:   picoshare.NeverExpire,
	}
}

func assertGuestUploadRows(t *testing.T, db *sql.DB, id picoshare.EntryID, wantEntries, wantChunks int) {
	t.Helper()
	var entries, chunks int
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM entries WHERE id = ?), (SELECT COUNT(*) FROM entries_data WHERE id = ?)`, id.String(), id.String()).Scan(&entries, &chunks); err != nil {
		t.Fatalf("failed to count upload rows: %v", err)
	}
	if got, want := entries, wantEntries; got != want {
		t.Errorf("entry %s metadata rows=%d, want=%d", id, got, want)
	}
	if got, want := chunks, wantChunks; got != want {
		t.Errorf("entry %s chunk rows=%d, want=%d", id, got, want)
	}
}
