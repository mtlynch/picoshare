package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

// PicoShare runs one process per database. Serialize its short reservation
// operations across store handles: SQLite shared-cache connections otherwise
// return SQLITE_LOCKED instead of waiting for a competing writer. Never hold
// this lock while receiving upload bytes.
var guestUploadDatabaseMu sync.Mutex

func (s Store) ReserveGuestUpload(ctx context.Context, id picoshare.GuestLinkID) (reservation picoshare.GuestUploadReservation, err error) {
	tx, err := s.beginGuestUploadTransaction(ctx)
	if err != nil {
		return reservation, err
	}
	defer rollbackGuestUpload(tx, &err)

	link, occupied, err := readGuestUploadPolicy(ctx, tx, id, picoshare.EntryID{})
	if err != nil {
		return reservation, err
	}
	if err := s.authorizeGuestUpload(link, occupied); err != nil {
		return reservation, err
	}
	reservation = picoshare.NewGuestUploadReservation(link)
	result, err := tx.ExecContext(ctx, `
		INSERT INTO guest_upload_reservations
			(entry_id, guest_link_id, max_file_bytes)
		SELECT :entry_id, :guest_link_id, :max_file_bytes
		WHERE NOT EXISTS (SELECT 1 FROM entries WHERE id = :entry_id)
			AND NOT EXISTS (SELECT 1 FROM entries_data WHERE id = :entry_id)`,
		sql.Named("entry_id", reservation.ID().String()),
		sql.Named("guest_link_id", id.String()),
		sql.Named("max_file_bytes", link.MaxFileBytes))
	if err != nil {
		return reservation, fmt.Errorf("failed to insert guest upload reservation: %w", err)
	}
	if n, err := result.RowsAffected(); err != nil {
		return reservation, fmt.Errorf("failed to count inserted guest upload reservations: %w", err)
	} else if n != 1 {
		return reservation, fmt.Errorf("failed to reserve guest upload: generated entry ID already owns data")
	}
	return reservation, tx.Commit()
}

// beginGuestUploadTransaction acquires the write lock before reading policy.
// This avoids upgrading a stale read transaction under competing admissions.
func (s Store) beginGuestUploadTransaction(ctx context.Context) (*sql.Tx, error) {
	guestUploadDatabaseMu.Lock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		guestUploadDatabaseMu.Unlock()
		return nil, fmt.Errorf("failed to begin guest upload transaction: %w", err)
	}
	if _, err = tx.ExecContext(ctx, "UPDATE guest_upload_reservations SET state = state WHERE 0"); err != nil {
		rollbackErr := tx.Rollback()
		guestUploadDatabaseMu.Unlock()
		return nil, errors.Join(fmt.Errorf("failed to acquire guest upload write lock: %w", err), rollbackErr)
	}
	return tx, nil
}

func rollbackGuestUpload(tx *sql.Tx, err *error) {
	defer guestUploadDatabaseMu.Unlock()
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		*err = errors.Join(*err, fmt.Errorf("failed to roll back guest upload transaction: %w", rollbackErr))
	}
}

func readGuestUploadPolicy(ctx context.Context, tx *sql.Tx, id picoshare.GuestLinkID, exclude picoshare.EntryID) (picoshare.GuestLink, int, error) {
	link, err := guestLinkFromRow(tx.QueryRowContext(ctx, `
		SELECT id, label, is_disabled, max_file_bytes, max_file_uploads,
			creation_time, url_expiration_time, file_expiration_time,
			(SELECT COUNT(*) FROM entries WHERE guest_link_id = :id),
			(SELECT COUNT(*) FROM guest_upload_reservations
			 WHERE guest_link_id = :id AND entry_id != :exclude)
		FROM guest_links WHERE id = :id`,
		sql.Named("id", id.String()), sql.Named("exclude", exclude.String())))
	var missing store.GuestLinkNotFoundError
	if errors.As(err, &missing) {
		return link, 0, store.GuestLinkNotFoundError{ID: id}
	} else if err != nil {
		return link, 0, fmt.Errorf("failed to read guest upload policy: %w", err)
	}
	return link, link.FilesUploaded + link.UploadsReserved, nil
}

func (s Store) authorizeGuestUpload(link picoshare.GuestLink, occupied int) error {
	if link.IsDisabled {
		return store.GuestUploadRejectedError{Reason: "guest link is disabled"}
	}
	if link.UrlExpires != picoshare.NeverExpire && !s.now().Before(time.Time(link.UrlExpires)) {
		return store.GuestUploadRejectedError{Reason: "guest link has expired"}
	}
	if link.MaxFileUploads != nil && occupied >= *link.MaxFileUploads {
		return store.GuestUploadRejectedError{Reason: "guest link has reached its upload limit"}
	}
	return nil
}

func (s Store) WriteGuestUpload(ctx context.Context, reservation picoshare.GuestUploadReservation, reader io.Reader) error {
	maximum, err := s.claimGuestUploadWriter(ctx, reservation)
	if err != nil {
		return err
	}

	// Deliberately keep chunk writes outside a transaction spanning the upload:
	// a long payload transaction bloats memory. Reservations retain ownership of
	// incomplete chunks until abort or startup recovery removes them.
	// See: https://github.com/mtlynch/picoshare/issues/284
	buffer := make([]byte, min(s.chunkSize, uint64(math.MaxInt)))
	var received uint64
	var chunkIndex int
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		readBuffer := buffer
		if maximum.Valid {
			remaining := uint64(maximum.Int64) - received
			// Probe one byte past the allowance, but never persist the probe.
			readBuffer = buffer[:min(uint64(len(buffer)), remaining+1)]
		}
		n, readErr := readGuestUploadChunk(ctx, reader, readBuffer)
		if maximum.Valid && uint64(n) > uint64(maximum.Int64)-received {
			return store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload exceeds its maximum file size of %d bytes", maximum.Int64)}
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return store.GuestUploadInvalidError{Err: fmt.Errorf("failed to read guest upload: %w", readErr)}
		}
		if n > 0 {
			result, err := s.execGuestUpload(ctx, `
				INSERT INTO entries_data (id, chunk_index, chunk)
				SELECT :id, :chunk_index, :chunk
				WHERE EXISTS (
					SELECT 1 FROM guest_upload_reservations
					WHERE entry_id = :id AND guest_link_id = :guest_link_id
						AND state = 'writing'
				) AND NOT EXISTS (SELECT 1 FROM entries WHERE id = :id)`,
				sql.Named("id", reservation.ID().String()),
				sql.Named("guest_link_id", reservation.GuestLink().ID.String()),
				sql.Named("chunk_index", chunkIndex), sql.Named("chunk", buffer[:n]))
			if err != nil {
				return fmt.Errorf("failed to write guest upload chunk: %w", err)
			}
			if n, err := result.RowsAffected(); err != nil {
				return fmt.Errorf("failed to count guest upload chunks: %w", err)
			} else if n != 1 {
				return store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload reservation no longer owns its chunks")}
			}
			received += uint64(n)
			chunkIndex++
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	if received == 0 {
		return store.GuestUploadInvalidError{Err: picoshare.ErrEmptyFile}
	}
	result, err := s.execGuestUpload(ctx, `
		UPDATE guest_upload_reservations SET state = 'written'
		WHERE entry_id = :id AND guest_link_id = :guest_link_id
			AND state = 'writing'`,
		sql.Named("id", reservation.ID().String()),
		sql.Named("guest_link_id", reservation.GuestLink().ID.String()))
	if err != nil {
		return fmt.Errorf("failed to finish writing guest upload: %w", err)
	}
	if n, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to count written guest upload reservations: %w", err)
	} else if n != 1 {
		return store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload reservation disappeared while writing")}
	}
	return nil
}

// readGuestUploadChunk fills a chunk while preserving the reader's EOF error.
// io.ReadFull turns a normal final EOF into ErrUnexpectedEOF, obscuring the
// distinction between a complete file and a truncated multipart body.
func readGuestUploadChunk(ctx context.Context, reader io.Reader, buffer []byte) (int, error) {
	var read int
	for read < len(buffer) {
		if err := ctx.Err(); err != nil {
			return read, err
		}
		n, err := reader.Read(buffer[read:])
		read += n
		if err != nil {
			return read, err
		}
	}
	return read, nil
}

func (s Store) claimGuestUploadWriter(ctx context.Context, reservation picoshare.GuestUploadReservation) (sql.NullInt64, error) {
	guestUploadDatabaseMu.Lock()
	defer guestUploadDatabaseMu.Unlock()
	var maximum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		UPDATE guest_upload_reservations SET state = 'writing'
		WHERE entry_id = :id AND guest_link_id = :guest_link_id
			AND state = 'reserved'
		RETURNING max_file_bytes`,
		sql.Named("id", reservation.ID().String()),
		sql.Named("guest_link_id", reservation.GuestLink().ID.String())).Scan(&maximum)
	if errors.Is(err, sql.ErrNoRows) {
		return maximum, store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload reservation is missing or already written")}
	} else if err != nil {
		return maximum, fmt.Errorf("failed to claim guest upload writer: %w", err)
	}
	return maximum, nil
}

func (s Store) execGuestUpload(ctx context.Context, query string, args ...any) (sql.Result, error) {
	guestUploadDatabaseMu.Lock()
	defer guestUploadDatabaseMu.Unlock()
	return s.db.ExecContext(ctx, query, args...)
}

func (s Store) CompleteGuestUpload(ctx context.Context, reservation picoshare.GuestUploadReservation, metadata picoshare.UploadMetadata) (err error) {
	tx, err := s.beginGuestUploadTransaction(ctx)
	if err != nil {
		return err
	}
	defer rollbackGuestUpload(tx, &err)
	var maximum sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT max_file_bytes FROM guest_upload_reservations
		WHERE entry_id = :id AND guest_link_id = :guest_link_id
			AND state = 'written'`,
		sql.Named("id", reservation.ID().String()),
		sql.Named("guest_link_id", reservation.GuestLink().ID.String())).Scan(&maximum)
	if errors.Is(err, sql.ErrNoRows) {
		return store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload reservation is missing or not fully written")}
	} else if err != nil {
		return fmt.Errorf("failed to read guest upload reservation: %w", err)
	}
	link, occupied, err := readGuestUploadPolicy(ctx, tx, reservation.GuestLink().ID, reservation.ID())
	var missing store.GuestLinkNotFoundError
	if errors.As(err, &missing) {
		return store.GuestUploadRejectedError{Reason: "guest link no longer exists"}
	}
	if err != nil {
		return err
	}
	if err := s.authorizeGuestUpload(link, occupied); err != nil {
		return err
	}
	var size uint64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(LENGTH(chunk)), 0)
		FROM entries_data WHERE id = ?`, reservation.ID().String()).Scan(&size); err != nil {
		return fmt.Errorf("failed to measure guest upload: %w", err)
	}
	if _, err := picoshare.FileSizeFromUint64(size); err != nil {
		return store.GuestUploadInvalidError{Err: err}
	}
	if maximum.Valid && size > uint64(maximum.Int64) {
		return store.GuestUploadInvalidError{Err: fmt.Errorf("guest upload exceeds its reserved file size")}
	}
	if link.MaxFileBytes != nil && size > *link.MaxFileBytes {
		return store.GuestUploadRejectedError{Reason: "guest upload exceeds the link's current maximum file size"}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO entries
			(id, guest_link_id, filename, note, content_type, upload_time,
			 expiration_time, download_passphrase)
		VALUES (:id, :guest_link_id, :filename, :note, :content_type,
			:upload_time, :expiration_time, :download_passphrase)`,
		sql.Named("id", reservation.ID().String()), sql.Named("guest_link_id", link.ID.String()),
		sql.Named("filename", metadata.Filename), sql.Named("note", metadata.Note.Value),
		sql.Named("content_type", metadata.ContentType), sql.Named("upload_time", formatTime(metadata.Uploaded)),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
		sql.Named("download_passphrase", downloadPassphraseString(metadata.DownloadPassphrase))); err != nil {
		return fmt.Errorf("failed to insert completed guest upload: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM guest_upload_reservations
		WHERE entry_id = ?`, reservation.ID().String()); err != nil {
		return fmt.Errorf("failed to release completed guest upload reservation: %w", err)
	}
	return tx.Commit()
}

func (s Store) AbortGuestUpload(reservation picoshare.GuestUploadReservation) error {
	return s.removeGuestUploads(&reservation)
}

// RecoverGuestUploads removes reservations abandoned by the previous process.
// Run it only at startup, before serving requests.
func (s Store) RecoverGuestUploads() error {
	return s.removeGuestUploads(nil)
}

// removeGuestUploads keeps capacity occupied until staged chunks are gone.
// Recovery is only safe before serving requests; abort targets one owner.
func (s Store) removeGuestUploads(reservation *picoshare.GuestUploadReservation) (err error) {
	ctx := context.Background()
	tx, err := s.beginGuestUploadTransaction(ctx)
	if err != nil {
		return err
	}
	defer rollbackGuestUpload(tx, &err)
	var id, guestLinkID any
	if reservation != nil {
		id, guestLinkID = reservation.ID().String(), reservation.GuestLink().ID.String()
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM entries_data
		WHERE id IN (
			SELECT entry_id FROM guest_upload_reservations
			WHERE (:id IS NULL OR entry_id = :id)
				AND (:guest_link_id IS NULL OR guest_link_id = :guest_link_id)
		) AND NOT EXISTS (
			SELECT 1 FROM entries WHERE entries.id = entries_data.id
		)`,
		sql.Named("id", id), sql.Named("guest_link_id", guestLinkID)); err != nil {
		return fmt.Errorf("failed to delete abandoned guest upload chunks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM guest_upload_reservations
		WHERE (:id IS NULL OR entry_id = :id)
			AND (:guest_link_id IS NULL OR guest_link_id = :guest_link_id)`,
		sql.Named("id", id), sql.Named("guest_link_id", guestLinkID)); err != nil {
		return fmt.Errorf("failed to delete abandoned guest upload reservations: %w", err)
	}
	return tx.Commit()
}
