package sqlite_test

import (
	"context"
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
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

type failedGuestUploadReader struct {
	err error
}

func TestGuestUploadLifecycle(t *testing.T) {
	db := test_sqlite.NewWithChunkSize(t, 2)
	link := picoshare.GuestLink{
		ID: "guest", Created: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		UrlExpires: picoshare.NeverExpire, MaxFileLifetime: picoshare.FileLifetimeInfinite,
		MaxFileUploads: new(1), MaxFileBytes: new(uint64(5)),
	}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err == nil {
		t.Fatal("second reservation must not acquire occupied capacity")
	}
	if err := db.CompleteGuestUpload(context.Background(), reservation, picoshare.UploadMetadata{}); err == nil {
		t.Fatal("unwritten reservation must not complete")
	}
	if err := db.WriteGuestUpload(context.Background(), reservation, strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	if err := db.WriteGuestUpload(context.Background(), reservation, strings.NewReader("again")); err == nil {
		t.Fatal("reservation must not accept a second file")
	}
	if err := db.CompleteGuestUpload(context.Background(), reservation, picoshare.UploadMetadata{
		ID: picoshare.MustCreateEntryID("wrongentry"), GuestLink: picoshare.GuestLink{ID: "wrong"},
		Filename: "hello.txt", Uploaded: link.Created, Expires: picoshare.NeverExpire,
		Size: mustParseFileSize(999),
	}); err != nil {
		t.Fatal(err)
	}
	metadata, err := db.GetEntryMetadata(reservation.ID())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := metadata.Size.UInt64(), uint64(5); got != want {
		t.Errorf("size=%d, want=%d", got, want)
	}
	if got, want := metadata.GuestLink.ID, link.ID; got != want {
		t.Errorf("guest link=%s, want=%s", got, want)
	}
	if err := db.AbortGuestUpload(reservation); err != nil {
		t.Fatal(err)
	}
	if err := db.RecoverGuestUploads(); err != nil {
		t.Fatal(err)
	}
	reader, err := db.ReadEntryFile(reservation.ID())
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "hello"; got != want {
		t.Errorf("content=%q, want=%q", got, want)
	}
	if err := db.CompleteGuestUpload(context.Background(), reservation, metadata); err == nil {
		t.Fatal("completed reservation must not complete again")
	}
	var rejected store.GuestUploadRejectedError
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); !errors.As(err, &rejected) {
		t.Fatalf("completed file must retain occupied capacity, admission error=%v", err)
	}
	if err := db.DeleteEntry(reservation.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
		t.Fatalf("deleting a completed file must restore capacity: %v", err)
	}
}

func TestInsertEntryRejectsGuestUploads(t *testing.T) {
	db := test_sqlite.New(t)
	err := db.InsertEntry(strings.NewReader("hello"), picoshare.UploadMetadata{
		ID: picoshare.MustCreateEntryID("abcdefghij"), GuestLink: picoshare.GuestLink{ID: "guest"},
		Uploaded: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), Expires: picoshare.NeverExpire,
	})
	var invalid store.GuestUploadInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("error=%v, want GuestUploadInvalidError", err)
	}
}

func TestReserveGuestUploadAdmission(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		explanation string
		missing     bool
		disabled    bool
		expires     picoshare.ExpirationTime
		limit       picoshare.GuestUploadCountLimit
		allowed     bool
	}{
		{
			explanation: "missing links return a missing-link error",
			missing:     true,
			expires:     picoshare.NeverExpire,
		},
		{
			explanation: "disabled links reject admission",
			disabled:    true,
			expires:     picoshare.NeverExpire,
		},
		{
			explanation: "links expiring now reject admission",
			expires:     picoshare.ExpirationTime(now),
		},
		{
			explanation: "future expiration permits admission",
			expires:     picoshare.ExpirationTime(now.Add(time.Hour)),
			allowed:     true,
		},
		{
			explanation: "unlimited upload counts permit admission",
			expires:     picoshare.NeverExpire,
			allowed:     true,
		},
		{
			explanation: "one remaining slot permits admission",
			expires:     picoshare.NeverExpire,
			limit:       new(1),
			allowed:     true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			db := test_sqlite.New(t)
			link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: tt.expires,
				MaxFileLifetime: picoshare.FileLifetimeInfinite, IsDisabled: tt.disabled, MaxFileUploads: tt.limit}
			if !tt.missing {
				if err := db.InsertGuestLink(link); err != nil {
					t.Fatal(err)
				}
			}
			reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
			switch {
			case tt.allowed:
				if err != nil {
					t.Fatal(err)
				}
				if got, want := reservation.GuestLink().ID, link.ID; got != want {
					t.Errorf("guest link=%s, want=%s", got, want)
				}
			case tt.missing:
				var missing store.GuestLinkNotFoundError
				if !errors.As(err, &missing) {
					t.Fatalf("error=%v, want GuestLinkNotFoundError", err)
				}
			default:
				var rejected store.GuestUploadRejectedError
				if !errors.As(err, &rejected) {
					t.Fatalf("error=%v, want GuestUploadRejectedError", err)
				}
			}
		})
	}
}

func TestGuestUploadConcurrentAdmissionAcrossHandles(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	db, _, path := createGuestUploadTestStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: new(1)}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	other := sqlite.New(sqlite.Params{Path: path, Now: func() time.Time { return now }})
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := db.ReserveGuestUpload(context.Background(), link.ID)
		results <- err
	}()
	go func() {
		<-start
		_, err := other.ReserveGuestUpload(context.Background(), link.ID)
		results <- err
	}()
	close(start)
	first, second := <-results, <-results
	if first != nil && second == nil {
		first, second = second, first
	}
	if first != nil {
		t.Fatalf("first admission=%v, want success", first)
	}
	var rejected store.GuestUploadRejectedError
	if !errors.As(second, &rejected) {
		t.Fatalf("second admission=%v, want GuestUploadRejectedError", second)
	}
}

func TestGuestUploadActualSizeAndReadErrors(t *testing.T) {
	readFailure := errors.New("reader disconnected")
	for _, tt := range []struct {
		explanation string
		limit       *uint64
		reader      io.Reader
		valid       bool
		underlying  error
	}{
		{
			explanation: "exact limit is accepted",
			limit:       new(uint64(5)),
			reader:      strings.NewReader("hello"),
			valid:       true,
		},
		{
			explanation: "one byte over the limit is rejected",
			limit:       new(uint64(5)),
			reader:      strings.NewReader("hello!"),
		},
		{
			explanation: "unlimited sizes accept larger payloads",
			reader:      strings.NewReader("hello, world!"),
			valid:       true,
		},
		{
			explanation: "empty files are rejected",
			reader:      strings.NewReader(""),
			underlying:  picoshare.ErrEmptyFile,
		},
		{
			explanation: "reader failures retain the underlying error",
			reader:      failedGuestUploadReader{err: readFailure},
			underlying:  readFailure,
		},
		{
			explanation: "unexpected reader EOF remains an invalid upload",
			reader:      failedGuestUploadReader{err: io.ErrUnexpectedEOF},
			underlying:  io.ErrUnexpectedEOF,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			db, raw, _ := createGuestUploadTestStore(t, func() time.Time { return now })
			link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileBytes: tt.limit}
			if err := db.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
			if err != nil {
				t.Fatal(err)
			}
			err = db.WriteGuestUpload(context.Background(), reservation, tt.reader)
			if tt.valid {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var invalid store.GuestUploadInvalidError
				if !errors.As(err, &invalid) {
					t.Fatalf("error=%v, want GuestUploadInvalidError", err)
				}
				if tt.underlying != nil && !errors.Is(err, tt.underlying) {
					t.Errorf("error=%v, want underlying=%v", err, tt.underlying)
				}
				if err := db.CompleteGuestUpload(context.Background(), reservation, picoshare.UploadMetadata{}); err == nil {
					t.Fatal("failed write must not complete")
				}
			}
			var stored uint64
			if err := raw.QueryRow("SELECT COALESCE(SUM(LENGTH(chunk)), 0) FROM entries_data WHERE id = ?", reservation.ID().String()).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if tt.limit != nil && stored > *tt.limit {
				t.Errorf("stored bytes=%d, exceeds allowance=%d", stored, *tt.limit)
			}
		})
	}
}

func TestAbortGuestUploadRetainsCapacityOnCleanupFailure(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		deleteTable string
	}{
		{
			explanation: "chunk deletion failure retains chunks and capacity",
			deleteTable: "entries_data",
		},
		{
			explanation: "reservation deletion failure rolls back chunk deletion",
			deleteTable: "guest_upload_reservations",
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			db, raw, _ := createGuestUploadTestStore(t, func() time.Time { return now })
			link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
				MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: new(1)}
			if err := db.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.WriteGuestUpload(context.Background(), reservation, strings.NewReader("hello")); err != nil {
				t.Fatal(err)
			}
			if _, err := raw.Exec(fmt.Sprintf(`CREATE TRIGGER fail_cleanup BEFORE DELETE ON %s
				BEGIN SELECT RAISE(ABORT, 'cleanup denied'); END`, tt.deleteTable)); err != nil {
				t.Fatal(err)
			}
			err = db.AbortGuestUpload(reservation)
			if err == nil || !strings.Contains(err.Error(), "cleanup denied") {
				t.Fatalf("cleanup error=%v, want underlying cleanup denied", err)
			}
			var invalid store.GuestUploadInvalidError
			var rejected store.GuestUploadRejectedError
			if errors.As(err, &invalid) || errors.As(err, &rejected) {
				t.Errorf("cleanup error=%v, want operational error", err)
			}
			if _, err := db.ReserveGuestUpload(context.Background(), link.ID); !errors.As(err, &rejected) {
				t.Fatalf("admission error=%v, want occupied-capacity rejection", err)
			}
			var stored int
			if err := raw.QueryRow("SELECT SUM(LENGTH(chunk)) FROM entries_data WHERE id = ?", reservation.ID().String()).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if got, want := stored, 5; got != want {
				t.Errorf("stored bytes=%d, want=%d", got, want)
			}
			if _, err := raw.Exec("DROP TRIGGER fail_cleanup"); err != nil {
				t.Fatal(err)
			}
			if err := db.AbortGuestUpload(reservation); err != nil {
				t.Fatal(err)
			}
			if err := raw.QueryRow("SELECT COUNT(*) FROM entries_data WHERE id = ?", reservation.ID().String()).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if got, want := stored, 0; got != want {
				t.Errorf("remaining chunks=%d, want=%d", got, want)
			}
			if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
				t.Fatalf("cleaned capacity must be reusable: %v", err)
			}
		})
	}
}

func TestRecoverGuestUploads(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	db, raw, path := createGuestUploadTestStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	completed, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WriteGuestUpload(context.Background(), completed, strings.NewReader("completed")); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteGuestUpload(context.Background(), completed, picoshare.UploadMetadata{
		Filename: "completed.txt", Uploaded: now, Expires: picoshare.NeverExpire,
	}); err != nil {
		t.Fatal(err)
	}
	abandoned, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WriteGuestUpload(context.Background(), abandoned, strings.NewReader("abandoned")); err != nil {
		t.Fatal(err)
	}
	other := sqlite.New(sqlite.Params{Path: path, Now: func() time.Time { return now }})
	var pending int
	if err := raw.QueryRow("SELECT COUNT(*) FROM guest_upload_reservations").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if got, want := pending, 1; got != want {
		t.Fatalf("opening another store leaves reservations=%d, want=%d", got, want)
	}
	if _, err := raw.Exec(`CREATE TRIGGER fail_recovery BEFORE DELETE ON guest_upload_reservations
		BEGIN SELECT RAISE(ABORT, 'recovery denied'); END`); err != nil {
		t.Fatal(err)
	}
	if err := other.RecoverGuestUploads(); err == nil || !strings.Contains(err.Error(), "recovery denied") {
		t.Fatalf("recovery error=%v, want underlying recovery denied", err)
	}
	var abandonedBytes int
	if err := raw.QueryRow("SELECT SUM(LENGTH(chunk)) FROM entries_data WHERE id = ?", abandoned.ID().String()).Scan(&abandonedBytes); err != nil {
		t.Fatal(err)
	}
	if got, want := abandonedBytes, len("abandoned"); got != want {
		t.Errorf("abandoned bytes after failed recovery=%d, want=%d", got, want)
	}
	if _, err := raw.Exec("DROP TRIGGER fail_recovery"); err != nil {
		t.Fatal(err)
	}
	if err := other.RecoverGuestUploads(); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow("SELECT COUNT(*) FROM guest_upload_reservations").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if got, want := pending, 0; got != want {
		t.Errorf("remaining reservations=%d, want=%d", got, want)
	}
	if err := raw.QueryRow("SELECT COUNT(*) FROM entries_data WHERE id = ?", abandoned.ID().String()).Scan(&abandonedBytes); err != nil {
		t.Fatal(err)
	}
	if got, want := abandonedBytes, 0; got != want {
		t.Errorf("abandoned chunks=%d, want=%d", got, want)
	}
	reader, err := db.ReadEntryFile(completed.ID())
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "completed"; got != want {
		t.Errorf("completed content=%q, want=%q", got, want)
	}
}

func TestCompleteGuestUploadRechecksPolicy(t *testing.T) {
	for _, tt := range []struct {
		explanation     string
		disable         bool
		deleteLink      bool
		expire          bool
		newSizeLimit    *uint64
		newCountLimit   *int
		competingUpload bool
	}{
		{
			explanation: "disabling a link rejects completion",
			disable:     true,
		},
		{
			explanation: "deleting a link rejects completion without deleting staged data",
			deleteLink:  true,
		},
		{
			explanation: "expiration during receiving rejects completion",
			expire:      true,
		},
		{
			explanation:  "a stricter size limit rejects an oversized staged file",
			newSizeLimit: new(uint64(4)),
		},
		{
			explanation:     "a stricter count limit includes other reservations",
			newCountLimit:   new(1),
			competingUpload: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			db, raw, _ := createGuestUploadTestStore(t, func() time.Time { return now })
			link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.ExpirationTime(now.Add(time.Hour)),
				MaxFileLifetime: picoshare.FileLifetimeInfinite}
			if err := db.InsertGuestLink(link); err != nil {
				t.Fatal(err)
			}
			reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.WriteGuestUpload(context.Background(), reservation, strings.NewReader("hello")); err != nil {
				t.Fatal(err)
			}
			if tt.competingUpload {
				if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
					t.Fatal(err)
				}
			}
			if tt.disable {
				if err := db.DisableGuestLink(link.ID); err != nil {
					t.Fatal(err)
				}
			}
			if tt.deleteLink {
				if err := db.DeleteGuestLink(link.ID); err != nil {
					t.Fatal(err)
				}
			}
			if tt.expire {
				now = now.Add(time.Hour)
			}
			if tt.newSizeLimit != nil {
				if _, err := raw.Exec("UPDATE guest_links SET max_file_bytes = ? WHERE id = ?", *tt.newSizeLimit, link.ID.String()); err != nil {
					t.Fatal(err)
				}
			}
			if tt.newCountLimit != nil {
				if _, err := raw.Exec("UPDATE guest_links SET max_file_uploads = ? WHERE id = ?", *tt.newCountLimit, link.ID.String()); err != nil {
					t.Fatal(err)
				}
			}
			err = db.CompleteGuestUpload(context.Background(), reservation, picoshare.UploadMetadata{
				Filename: "hello.txt", Uploaded: now, Expires: picoshare.NeverExpire,
			})
			var rejected store.GuestUploadRejectedError
			if !errors.As(err, &rejected) {
				t.Fatalf("completion error=%v, want GuestUploadRejectedError", err)
			}
			var stored int
			if err := raw.QueryRow("SELECT SUM(LENGTH(chunk)) FROM entries_data WHERE id = ?", reservation.ID().String()).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if got, want := stored, 5; got != want {
				t.Errorf("staged bytes after rejection=%d, want=%d", got, want)
			}
			if err := db.AbortGuestUpload(reservation); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPurgePreservesGuestUploadReservations(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	db, raw, _ := createGuestUploadTestStore(t, func() time.Time { return now })
	link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: new(1)}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WriteGuestUpload(context.Background(), reservation, strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	orphan := picoshare.MustCreateEntryID("orphanedid")
	if _, err := raw.Exec("INSERT INTO entries_data (id, chunk_index, chunk) VALUES (?, 0, ?)", orphan.String(), []byte("orphaned")); err != nil {
		t.Fatal(err)
	}
	if err := db.Purge(); err != nil {
		t.Fatal(err)
	}
	var orphanChunks int
	if err := raw.QueryRow("SELECT COUNT(*) FROM entries_data WHERE id = ?", orphan.String()).Scan(&orphanChunks); err != nil {
		t.Fatal(err)
	}
	if got, want := orphanChunks, 0; got != want {
		t.Errorf("orphan chunks=%d, want=%d", got, want)
	}
	var rejected store.GuestUploadRejectedError
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); !errors.As(err, &rejected) {
		t.Fatalf("purge must retain occupied capacity, admission error=%v", err)
	}
	if err := db.CompleteGuestUpload(context.Background(), reservation, picoshare.UploadMetadata{
		Filename: "hello.txt", Uploaded: now, Expires: picoshare.NeverExpire,
	}); err != nil {
		t.Fatal(err)
	}
	reader, err := db.ReadEntryFile(reservation.ID())
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "hello"; got != want {
		t.Errorf("completed content after purge=%q, want=%q", got, want)
	}
}

func TestGuestUploadReservationsAreIsolatedPerLink(t *testing.T) {
	db := test_sqlite.New(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	first := picoshare.GuestLink{ID: "first", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: new(1)}
	second := first
	second.ID = "second"
	if err := db.InsertGuestLink(first); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertGuestLink(second); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), second.ID); err != nil {
		t.Fatal(err)
	}
	link, err := db.GetGuestLink(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := link.FilesUploaded, 0; got != want {
		t.Errorf("completed files=%d, want=%d", got, want)
	}
	if got, want := link.UploadsReserved, 1; got != want {
		t.Errorf("reserved uploads=%d, want=%d", got, want)
	}
}

func TestGuestLinkQueriesCountEntriesAndReservationsIndependently(t *testing.T) {
	db := test_sqlite.New(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	completed, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WriteGuestUpload(context.Background(), completed, strings.NewReader("completed")); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteGuestUpload(context.Background(), completed, picoshare.UploadMetadata{
		Filename: "completed.txt", Uploaded: now, Expires: picoshare.NeverExpire,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
		t.Fatal(err)
	}
	individual, err := db.GetGuestLink(link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := individual.FilesUploaded, 1; got != want {
		t.Errorf("individual completed files=%d, want=%d", got, want)
	}
	if got, want := individual.UploadsReserved, 2; got != want {
		t.Errorf("individual reservations=%d, want=%d", got, want)
	}
	links, err := db.GetGuestLinks()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(links), 1; got != want {
		t.Fatalf("guest links=%d, want=%d", got, want)
	}
	if got, want := links[0].FilesUploaded, 1; got != want {
		t.Errorf("listed completed files=%d, want=%d", got, want)
	}
	if got, want := links[0].UploadsReserved, 2; got != want {
		t.Errorf("listed reservations=%d, want=%d", got, want)
	}
}

func TestCanceledGuestUploadCanAbort(t *testing.T) {
	db := test_sqlite.New(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	link := picoshare.GuestLink{ID: "guest", Created: now, UrlExpires: picoshare.NeverExpire,
		MaxFileLifetime: picoshare.FileLifetimeInfinite, MaxFileUploads: new(1)}
	if err := db.InsertGuestLink(link); err != nil {
		t.Fatal(err)
	}
	reservation, err := db.ReserveGuestUpload(context.Background(), link.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := db.WriteGuestUpload(ctx, reservation, strings.NewReader("hello")); !errors.Is(err, context.Canceled) {
		t.Fatalf("write error=%v, want context cancellation", err)
	}
	if err := db.CompleteGuestUpload(ctx, reservation, picoshare.UploadMetadata{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("completion error=%v, want context cancellation", err)
	}
	var rejected store.GuestUploadRejectedError
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); !errors.As(err, &rejected) {
		t.Fatalf("canceled upload must retain capacity, admission error=%v", err)
	}
	if err := db.AbortGuestUpload(reservation); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveGuestUpload(context.Background(), link.ID); err != nil {
		t.Fatalf("aborted upload must release capacity: %v", err)
	}
}

func (r failedGuestUploadReader) Read([]byte) (int, error) {
	return 0, r.err
}

func createGuestUploadTestStore(t *testing.T, now func() time.Time) (sqlite.Store, *sql.DB, string) {
	t.Helper()
	path := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", random.String(20, []rune("abcdefghijklmnopqrstuvwxyz")))
	db := sqlite.New(sqlite.Params{Path: path, Now: now, PicoShareChunkSize: 2})
	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Error(err)
		}
	})
	return db, raw, path
}
