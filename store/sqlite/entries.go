package sqlite

import (
	"context"
	"database/sql"
	"io"
	"log"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
)

func (s Store) GetEntriesMetadata() ([]picoshare.UploadMetadata, error) {
	rows, err := s.ctx.Query(`
	SELECT
		e.id,
		e.filename,
		e.note,
		e.content_type,
		e.upload_time,
		e.expiration_time,
		COALESCE(c.file_size, (
			SELECT SUM(LENGTH(chunk))
			FROM entries_data
			WHERE entries_data.id = e.id
		)) AS file_size,
		COALESCE(c.download_count, 0) AS download_count
	FROM
		entries e
	LEFT JOIN
		entry_cache c ON e.id = c.entry_id
	ORDER BY
		e.upload_time DESC`)
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
		var downloadCount uint64
		if err = rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSizeRaw, &downloadCount); err != nil {
			return []picoshare.UploadMetadata{}, err
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
			ID:            picoshare.EntryID(id),
			Filename:      picoshare.Filename(filename),
			Note:          picoshare.FileNote{Value: note},
			ContentType:   picoshare.ContentType(contentType),
			Uploaded:      ut,
			Expires:       picoshare.ExpirationTime(et),
			Size:          fileSize,
			DownloadCount: downloadCount,
		})
	}

	return ee, nil
}

func (s Store) ReadEntryFile(id picoshare.EntryID) (io.ReadSeeker, error) {
	r, err := file.NewReader(s.ctx, id)
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
	var fileSizeRaw uint64
	var guestLinkID *picoshare.GuestLinkID
	err := s.ctx.QueryRow(`
	SELECT
		e.filename,
		e.note,
		e.content_type,
		e.upload_time,
		e.expiration_time,
		COALESCE(c.file_size, (
			SELECT SUM(LENGTH(chunk))
			FROM entries_data
			WHERE entries_data.id = e.id
		)) AS file_size,
		e.guest_link_id
	FROM
		entries e
	LEFT JOIN
		entry_cache c ON e.id = c.entry_id
	WHERE
		e.id = :entry_id`, sql.Named("entry_id", id)).Scan(&filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSizeRaw, &guestLinkID)
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

	return picoshare.UploadMetadata{
		ID:          id,
		Filename:    picoshare.Filename(filename),
		GuestLink:   guestLink,
		Note:        picoshare.FileNote{Value: note},
		ContentType: picoshare.ContentType(contentType),
		Uploaded:    ut,
		Expires:     picoshare.ExpirationTime(et),
		Size:        fileSize,
	}, nil
}

func (s Store) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	// Note: We deliberately don't use a transaction here, as it bloats memory, so
	// we can end up in a state with orphaned entries data. We clean it up in
	// Purge().
	// See: https://github.com/mtlynch/picoshare/issues/284
	w := file.NewWriter(s.ctx, metadata.ID, s.chunkSize)
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	// Close() flushes the buffer, and it can fail.
	if err := w.Close(); err != nil {
		return err
	}

	// Calculate file size after writing the file data
	var fileSize uint64
	err := s.ctx.QueryRow(`
	SELECT
		SUM(LENGTH(chunk)) AS file_size
	FROM
		entries_data
	WHERE
		id = :entry_id`, sql.Named("entry_id", metadata.ID)).Scan(&fileSize)
	if err != nil {
		log.Printf("failed to calculate file size for entry %s: %v", metadata.ID, err)
		return err
	}

	// Insert entry metadata
	_, err = s.ctx.Exec(`
	INSERT INTO
		entries
	(
		id,
		guest_link_id,
		filename,
		note,
		content_type,
		upload_time,
		expiration_time
	)
	VALUES(:entry_id, NULLIF(:guest_link_id, ''), :filename, :note, :content_type, :upload_time, :expiration_time)`,
		sql.Named("entry_id", metadata.ID),
		sql.Named("guest_link_id", metadata.GuestLink.ID),
		sql.Named("filename", metadata.Filename),
		sql.Named("note", metadata.Note.Value),
		sql.Named("content_type", metadata.ContentType),
		sql.Named("upload_time", formatTime(metadata.Uploaded)),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
	)
	if err != nil {
		log.Printf("insert into entries table failed, aborting transaction: %v", err)
		return err
	}

	// Insert cache entry
	_, err = s.ctx.Exec(`
	INSERT INTO
		entry_cache
	(
		entry_id,
		file_size,
		download_count,
		last_updated
	)
	VALUES(:entry_id, :file_size, 0, :last_updated)`,
		sql.Named("entry_id", metadata.ID),
		sql.Named("file_size", fileSize),
		sql.Named("last_updated", formatTime(metadata.Uploaded)),
	)
	if err != nil {
		log.Printf("insert into entry_cache table failed: %v", err)
		return err
	}

	return nil
}

func (s Store) UpdateEntryMetadata(id picoshare.EntryID, metadata picoshare.UploadMetadata) error {
	log.Printf("updating metadata for entry %s", id)

	res, err := s.ctx.Exec(`
	UPDATE entries
	SET
		filename = :filename,
		expiration_time = :expiration_time,
		note = :note
	WHERE
		id = :entry_id`,
		sql.Named("filename", metadata.Filename),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
		sql.Named("note", metadata.Note.Value),
		sql.Named("entry_id", id))
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

	tx, err := s.ctx.BeginTx(context.Background(), nil)
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
		entry_id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		log.Printf("delete from downloads table failed, aborting transaction: %v", err)
		return err
	}

	if _, err := tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		log.Printf("delete from entries_data table failed, aborting transaction: %v", err)
		return err
	}

	// Delete cache entry (will be automatically deleted by foreign key cascade, but explicit is clearer)
	if _, err := tx.Exec(`
	DELETE FROM
		entry_cache
	WHERE
		entry_id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		log.Printf("delete from entry_cache table failed, aborting transaction: %v", err)
		return err
	}

	if _, err := tx.Exec(`
	DELETE FROM
		entries
	WHERE
		id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		log.Printf("delete from entries table failed, aborting transaction: %v", err)
		return err
	}

	return tx.Commit()
}

// RebuildEntryCache rebuilds the entire entry cache from scratch
func (s Store) RebuildEntryCache() error {
	log.Printf("rebuilding entry cache")

	tx, err := s.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback cache rebuild: %v", err)
		}
	}()

	// Clear existing cache
	if _, err := tx.Exec(`DELETE FROM entry_cache`); err != nil {
		return err
	}

	// Rebuild cache
	_, err = tx.Exec(`
	INSERT INTO entry_cache (entry_id, file_size, download_count, last_updated)
	SELECT
		e.id,
		COALESCE(fs.file_size, 0) as file_size,
		COALESCE(dc.download_count, 0) as download_count,
		datetime('now') as last_updated
	FROM entries e
	LEFT JOIN (
		SELECT id, SUM(LENGTH(chunk)) as file_size
		FROM entries_data
		GROUP BY id
	) fs ON e.id = fs.id
	LEFT JOIN (
		SELECT entry_id, COUNT(*) as download_count
		FROM downloads
		GROUP BY entry_id
	) dc ON e.id = dc.entry_id`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
