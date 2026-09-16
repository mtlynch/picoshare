package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/sqlite/file"
)

func (s Store) GetEntriesMetadata() ([]picoshare.UploadMetadata, error) {
	rows, err := s.db.Query(`
	SELECT
		entries.id AS id,
		entries.filename AS filename,
		entries.note AS note,
		entries.content_type AS content_type,
		entries.upload_time AS upload_time,
		entries.expiration_time AS expiration_time,
		sizes.file_size AS file_size
	FROM
		entries
	INNER JOIN
		(
			SELECT
				id,
				SUM(LENGTH(chunk)) AS file_size
			FROM
				entries_data
			GROUP BY
				id
		) sizes ON entries.id = sizes.id`)
	if err != nil {
		return []picoshare.UploadMetadata{}, err
	}

	ee := []picoshare.UploadMetadata{}
	for rows.Next() {
		var id string
		var filename string
		var note *string
		var contentType string
		var uploadTimeRaw string
		var expirationTimeRaw string
		var fileSizeRaw uint64
		if err = rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSizeRaw); err != nil {
			return []picoshare.UploadMetadata{}, err
		}
		entryID, err := picoshare.EntryIDFromString(id)
		if err != nil {
			return []picoshare.UploadMetadata{}, fmt.Errorf("failed to parse entry ID from database: %w", err)
		}

		ut, err := parseDatetime(uploadTimeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		et, err := parseDatetime(expirationTimeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		fileSize, err := picoshare.FileSizeFromUint64(fileSizeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		ee = append(ee, picoshare.UploadMetadata{
			ID:          entryID,
			Filename:    picoshare.Filename(filename),
			Note:        picoshare.FileNote{Value: note},
			ContentType: picoshare.ContentType(contentType),
			Uploaded:    ut,
			Expires:     picoshare.ExpirationTime(et),
			Size:        fileSize,
		})
	}

	return ee, nil
}

func (s Store) ReadEntryFile(id picoshare.EntryID) (io.ReadSeeker, error) {
	r, err := file.NewReader(s.db, id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s Store) GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	var filename string
	var note *string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	var downloadPassphraseRaw *string
	var fileSizeRaw uint64
	var guestLinkID *picoshare.GuestLinkID
	err := s.db.QueryRow(`
	SELECT
		entries.filename AS filename,
		entries.note AS note,
		entries.content_type AS content_type,
		entries.upload_time AS upload_time,
		entries.expiration_time AS expiration_time,
		entries.download_passphrase AS download_passphrase,
		sizes.file_size AS file_size,
		entries.guest_link_id AS guest_link_id
	FROM
		entries
	INNER JOIN
		(
			SELECT
				id,
				SUM(LENGTH(chunk)) AS file_size
			FROM
				entries_data
			GROUP BY
				id
		) sizes ON entries.id = sizes.id
	WHERE
		entries.id = :entry_id`, sql.Named("entry_id", id.String())).Scan(&filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &downloadPassphraseRaw, &fileSizeRaw, &guestLinkID)
	if err == sql.ErrNoRows {
		return picoshare.UploadMetadata{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	var guestLink picoshare.GuestLink
	if guestLinkID != nil && !guestLinkID.Empty() {
		guestLink, err = s.GetGuestLink(*guestLinkID)
		if err != nil {
			return picoshare.UploadMetadata{}, err
		}
	}

	ut, err := parseDatetime(uploadTimeRaw)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	fileSize, err := picoshare.FileSizeFromUint64(fileSizeRaw)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}
	downloadPassphrase, err := parseDownloadPassphrase(downloadPassphraseRaw)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	return picoshare.UploadMetadata{
		ID:                 id,
		Filename:           picoshare.Filename(filename),
		GuestLink:          guestLink,
		Note:               picoshare.FileNote{Value: note},
		ContentType:        picoshare.ContentType(contentType),
		Uploaded:           ut,
		Expires:            picoshare.ExpirationTime(et),
		Size:               fileSize,
		DownloadPassphrase: downloadPassphrase,
	}, nil
}

func (s Store) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	// Note: We deliberately don't use a transaction here, as it bloats memory, so
	// we can end up in a state with orphaned entries data. We clean it up in
	// Purge().
	// See: https://github.com/mtlynch/picoshare/issues/284
	w := file.NewWriter(s.db, metadata.ID, s.chunkSize)
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	// Close() flushes the buffer, and it can fail.
	if err := w.Close(); err != nil {
		return err
	}

	_, err := s.db.Exec(`
	INSERT INTO
		entries
	(
		id,
		guest_link_id,
		filename,
		note,
		content_type,
		upload_time,
		expiration_time,
		download_passphrase
	)
	VALUES(:entry_id, NULLIF(:guest_link_id, ''), :filename, :note, :content_type, :upload_time, :expiration_time, :download_passphrase)`,
		sql.Named("entry_id", metadata.ID.String()),
		sql.Named("guest_link_id", metadata.GuestLink.ID),
		sql.Named("filename", metadata.Filename),
		sql.Named("note", metadata.Note.Value),
		sql.Named("content_type", metadata.ContentType),
		sql.Named("upload_time", formatTime(metadata.Uploaded)),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
		sql.Named("download_passphrase", downloadPassphraseString(metadata.DownloadPassphrase)),
	)
	if err != nil {
		log.Printf("insert into entries table failed, aborting transaction: %v", err)
		return err
	}

	return nil
}

func parseDownloadPassphrase(raw *string) (picoshare.DownloadPassphrase, error) {
	if raw == nil {
		return picoshare.DownloadPassphrase{}, nil
	}
	return picoshare.NewDownloadPassphrase(*raw)
}

func downloadPassphraseString(passphrase picoshare.DownloadPassphrase) *string {
	if passphrase.Empty() {
		return nil
	}
	s := passphrase.String()
	return &s
}

func (s Store) UpdateEntryMetadata(id picoshare.EntryID, metadata picoshare.UploadMetadata) error {
	log.Printf("updating metadata for entry %s", id)

	res, err := s.db.Exec(`
	UPDATE entries
	SET
		filename = :filename,
		expiration_time = :expiration_time,
		note = :note,
		download_passphrase = :download_passphrase
	WHERE
		id = :entry_id`,
		sql.Named("filename", metadata.Filename),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
		sql.Named("note", metadata.Note.Value),
		sql.Named("download_passphrase", downloadPassphraseString(metadata.DownloadPassphrase)),
		sql.Named("entry_id", id.String()))
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return store.EntryNotFoundError{ID: id}
	}

	return nil
}

func (s Store) DeleteEntry(id picoshare.EntryID) error {
	log.Printf("deleting entry %v", id)

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback delete entry: %v", err)
		}
	}()

	if _, err := tx.Exec(`
	DELETE FROM
		downloads
	WHERE
		entry_id = :entry_id`, sql.Named("entry_id", id.String())); err != nil {
		log.Printf("delete from downloads table failed, aborting transaction: %v", err)
		return err
	}

	if _, err := tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id = :entry_id`, sql.Named("entry_id", id.String())); err != nil {
		log.Printf("delete from entries_data table failed, aborting transaction: %v", err)
		return err
	}

	if _, err := tx.Exec(`
	DELETE FROM
		entries
	WHERE
		id = :entry_id`, sql.Named("entry_id", id.String())); err != nil {
		log.Printf("delete from entries table failed, aborting transaction: %v", err)
		return err
	}

	return tx.Commit()
}
