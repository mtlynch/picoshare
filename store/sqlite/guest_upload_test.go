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

func TestInsertEntryGuestUploadPolicy(t *testing.T) {
	for _, tt := range []struct {
		explanation            string
		initialMaxFileUploads  picoshare.GuestUploadCountLimit
		maxFileBytes           picoshare.GuestUploadMaxFileBytes
		maxFileUploads         picoshare.GuestUploadCountLimit
		hasPriorUploadForLink  bool
		hasPriorUploadForOther bool
		payload                string
		reportedSize           int
		wantRejected           bool
	}{
		{
			explanation:           "reducing the upload limit to the existing count rejects the staged upload",
			initialMaxFileUploads: new(2),
			maxFileUploads:        new(1),
			hasPriorUploadForLink: true,
			payload:               "abcdefgh",
			reportedSize:          8,
			wantRejected:          true,
		},
		{
			explanation:    "the last available upload slot accepts an upload",
			maxFileUploads: new(1),
			payload:        "abcdefgh",
			reportedSize:   8,
		},
		{
			explanation:            "uploads through another link do not consume this link's capacity",
			maxFileUploads:         new(1),
			hasPriorUploadForOther: true,
			payload:                "abcdefgh",
			reportedSize:           8,
		},
		{
			explanation:           "unlimited upload count accepts another upload",
			hasPriorUploadForLink: true,
			payload:               "abcdefgh",
			reportedSize:          8,
		},
		{
			explanation:  "unlimited file size accepts a payload larger than its reported size",
			payload:      "abcdefgh",
			reportedSize: 1,
		},
		{
			explanation:  "a payload exactly at the byte limit succeeds",
			maxFileBytes: new(uint64(8)),
			payload:      "abcdefgh",
			reportedSize: 8,
		},
		{
			explanation:  "reducing the byte limit below the staged payload size rejects the upload",
			maxFileBytes: new(uint64(7)),
			payload:      "abcdefgh",
			reportedSize: 8,
			wantRejected: true,
		},
		{
			explanation:  "underreported metadata cannot bypass the actual byte limit",
			maxFileBytes: new(uint64(7)),
			payload:      "abcdefgh",
			reportedSize: 1,
			wantRejected: true,
		},
		{
			explanation:  "overreported metadata does not reject a payload within the byte limit",
			maxFileBytes: new(uint64(8)),
			payload:      "abcdefgh",
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
				MaxFileUploads:  tt.initialMaxFileUploads,
			}
			if err := dataStore.InsertGuestLink(link); err != nil {
				t.Fatalf("failed to insert guest link: %v", err)
			}
			wantUploadCount := 1
			if tt.hasPriorUploadForLink {
				metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("priorentry"), link, now)
				if err := dataStore.InsertEntry(strings.NewReader("prior"), metadata); err != nil {
					t.Fatalf("failed to insert prior upload: %v", err)
				}
				wantUploadCount++
			}
			if tt.hasPriorUploadForOther {
				otherLink := link
				otherLink.ID = "other-link"
				if err := dataStore.InsertGuestLink(otherLink); err != nil {
					t.Fatalf("failed to insert other guest link: %v", err)
				}
				metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("otherentry"), otherLink, now)
				if err := dataStore.InsertEntry(strings.NewReader("other"), metadata); err != nil {
					t.Fatalf("failed to insert upload for other link: %v", err)
				}
			}
			// The upload retains the original link snapshot, not its current policy.
			metadata := createGuestUploadMetadata(picoshare.MustCreateEntryID("newentryab"), link, now)
			metadata.Size = mustParseFileSize(tt.reportedSize)
			policyApplied := false
			reader := io.MultiReader(strings.NewReader(tt.payload), guestUploadReadFunc(func([]byte) (int, error) {
				// Both full chunks have been persisted before the policy changes.
				assertGuestUploadRows(t, db, metadata.ID, 0, 2)
				if _, err := db.Exec(`UPDATE guest_links SET max_file_bytes = ?, max_file_uploads = ? WHERE id = ?`,
					tt.maxFileBytes, tt.maxFileUploads, link.ID); err != nil {
					t.Fatalf("failed to set guest link limits during payload read: %v", err)
				}
				policyApplied = true
				return 0, io.EOF
			}))
			err := dataStore.InsertEntry(reader, metadata)
			if got, want := policyApplied, true; got != want {
				t.Fatalf("final policy applied=%v, want=%v", got, want)
			}
			if got, want := err != nil, tt.wantRejected; got != want {
				t.Fatalf("upload rejected=%v, want=%v (err=%v)", got, want, err)
			}
			if tt.wantRejected {
				if got, want := errors.Is(err, store.ErrGuestUploadRejected), true; got != want {
					t.Errorf("policy rejection=%v, want=%v (err=%v)", got, want, err)
				}
				assertGuestUploadRows(t, db, metadata.ID, 0, 0)
				return
			}
			assertGuestUploadRows(t, db, metadata.ID, 1, 2)
			stored, err := dataStore.GetEntryMetadata(metadata.ID)
			if err != nil {
				t.Fatalf("failed to read accepted upload metadata: %v", err)
			}
			if got, want := stored.Size, mustParseFileSize(len(tt.payload)); !got.Equal(want) {
				t.Errorf("stored size=%v, want=%v", got, want)
			}
			if got, want := stored.GuestLink.FilesUploaded, wantUploadCount; got != want {
				t.Errorf("link upload count=%d, want=%d", got, want)
			}
		})
	}
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
