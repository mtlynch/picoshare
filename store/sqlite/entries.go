package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log"

	"github.com/ncruces/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

func (s Store) GetEntriesMetadata() ([]picoshare.UploadMetadata, error) {
	rows, err := s.ctx.Query(`
	SELECT
		entries.id AS id,
		entries.filename AS filename,
		entries.note AS note,
		entries.content_type AS content_type,
		entries.upload_time AS upload_time,
		entries.expiration_time AS expiration_time,
		IFNULL(downloads.download_count, 0) AS download_count
	FROM
		entries
	LEFT OUTER JOIN
		(
			SELECT
				entry_id,
				COUNT (entry_id) as download_count
			FROM
				downloads
			GROUP BY
				entry_id
		) downloads ON entries.id = downloads.entry_id`)
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
		var downloadCount uint64
		if err = rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &downloadCount); err != nil {
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

		ee = append(ee, picoshare.UploadMetadata{
			ID:            picoshare.EntryID(id),
			Filename:      picoshare.Filename(filename),
			Note:          picoshare.FileNote{Value: note},
			ContentType:   picoshare.ContentType(contentType),
			Uploaded:      ut,
			Expires:       picoshare.ExpirationTime(et),
			Size:          13, // TODO: Calculate this for real
			DownloadCount: downloadCount,
		})
	}

	return ee, nil
}

func (s Store) GetEntry(id picoshare.EntryID) (picoshare.UploadEntry, error) {
	metadata, err := s.GetEntryMetadata(id)
	if err != nil {
		return picoshare.UploadEntry{}, err
	}

	var rowid int
	err = s.ctx.QueryRow(`
	SELECT
		rowid
	FROM
		entries
	WHERE
		entries.id = :entry_id`, sql.Named("entry_id", id)).Scan(&rowid)
	if err == sql.ErrNoRows {
		return picoshare.UploadEntry{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.UploadEntry{}, err
	}

	log.Printf("rowid=%v", rowid) // DEBUG

	var buf bytes.Buffer
	reader := bytes.NewReader(buf.Bytes())

	_, err = s.ctx.Exec(
		`SELECT readblob('main', 'entries', 'contents', :id, :offset, :reader)`,
		sql.Named("id", rowid),
		sql.Named("offset", 0),
		sql.Named("reader", sqlite3.Pointer(reader)))
	if err != nil {
		return picoshare.UploadEntry{}, err
	}

	return picoshare.UploadEntry{
		UploadMetadata: metadata,
		Reader:         reader,
	}, nil
}

func (s Store) GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	var filename string
	var note *string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	var guestLinkID *picoshare.GuestLinkID
	err := s.ctx.QueryRow(`
	SELECT
		entries.filename AS filename,
		entries.note AS note,
		entries.content_type AS content_type,
		entries.upload_time AS upload_time,
		entries.expiration_time AS expiration_time,
		entries.guest_link_id AS guest_link_id
	FROM
		entries
	WHERE
		entries.id = :entry_id`, sql.Named("entry_id", id)).Scan(&filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &guestLinkID)
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

	return picoshare.UploadMetadata{
		ID:          id,
		Filename:    picoshare.Filename(filename),
		GuestLink:   guestLink,
		Note:        picoshare.FileNote{Value: note},
		ContentType: picoshare.ContentType(contentType),
		Uploaded:    ut,
		Expires:     picoshare.ExpirationTime(et),
		Size:        13, // TODO: Calculate for real
	}, nil
}

func (s Store) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	_, err := s.ctx.Exec(`
	INSERT INTO
		entries
	(
		id,
		guest_link_id,
		filename,
		contents,
		note,
		content_type,
		upload_time,
		expiration_time
	)
	VALUES(:entry_id, :guest_link_id, :filename, :contents, :note, :content_type, :upload_time, :expiration_time)`,
		sql.Named("entry_id", metadata.ID),
		sql.Named("guest_link_id", nil),
		sql.Named("filename", metadata.Filename),
		sql.Named("contents", sqlite3.ZeroBlob(metadata.Size)),
		sql.Named("note", metadata.Note.Value),
		sql.Named("content_type", metadata.ContentType),
		sql.Named("upload_time", formatTime(metadata.Uploaded)),
		sql.Named("expiration_time", formatExpirationTime(metadata.Expires)),
	)
	if err != nil {
		log.Printf("insert into entries table failed, aborting transaction: %v", err)
		return err
	}

	log.Printf("created entry row for %s", metadata.ID) // DEBUG

	_, err = s.ctx.Exec(`SELECT writeblob('main', 'entries', 'contents', last_insert_rowid(), :offset, :data)`,
		sql.Named("offset", 0),
		sql.Named("data", sqlite3.Pointer(reader)))
	if err != nil {
		log.Printf("failed to write blob of length %d: %v", metadata.Size, err)
		return err
	}

	log.Printf("wrote blob of size %d for %v", metadata.Size, metadata.ID) // DEBUG

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

	if _, err := tx.Exec(`
	DELETE FROM
		entries
	WHERE
		id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		log.Printf("delete from entries table failed, aborting transaction: %v", err)
		return err
	}

	// TODO: No transaction

	return tx.Commit()
}
