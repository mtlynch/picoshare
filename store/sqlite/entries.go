package sqlite

import (
	"database/sql"
	"fmt"
	"io"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/blobio"

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
		LENGTH(entries.contents) AS file_size,
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
		var fileSize uint64
		var downloadCount uint64
		if err = rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSize, &downloadCount); err != nil {
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
			Size:          fileSize,
			DownloadCount: downloadCount,
		})
	}

	return ee, nil
}

func (s Store) ReadEntryFile(id picoshare.EntryID, processFile func(io.ReadSeeker)) error {
	log.Printf("attempting to read entry %s", id.String()) // DEBUG
	_, err := s.ctx.Exec(`
			SELECT
				openblob('main', 'entries', 'contents', rowid, :writeMode, :callback)
			FROM
				entries
			WHERE
				entries.id = :entry_id
	`,
		sql.Named("writeMode", false),
		sql.Named("callback",
			sqlite3.Pointer[blobio.OpenCallback](func(blob *sqlite3.Blob, _ ...sqlite3.Value) error {
				log.Printf("start callback") // DEBUG
				processFile(blob)
				log.Printf("end callback") // DEBUG
				return nil
			})),
		sql.Named("entry_id", id))
	if err != nil {
		log.Printf("failed to open blob for %v: %v", id.String(), err) // DEBUG
		return fmt.Errorf("error opening blob for id %s: %w", id, err)
	}

	log.Printf("finished reading entry %v", id.String()) // DEBUG

	return nil
}

func (s Store) GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	var filename string
	var note *string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	var fileSize uint64
	var guestLinkID *picoshare.GuestLinkID
	err := s.ctx.QueryRow(`
	SELECT
		filename AS filename,
		note AS note,
		content_type AS content_type,
		upload_time AS upload_time,
		expiration_time AS expiration_time,
		LENGTH(contents) AS file_size,
		guest_link_id AS guest_link_id
	FROM
		entries
	WHERE
		id = :entry_id`,
		sql.Named("entry_id", id)).Scan(&filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSize, &guestLinkID)
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
		Size:        fileSize,
	}, nil
}

func (s Store) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	r, err := s.ctx.Exec(`
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
	VALUES(:entry_id, NULLIF(:guest_link_id, ''), :filename, :contents, :note, :content_type, :upload_time, :expiration_time)`,
		sql.Named("entry_id", metadata.ID),
		sql.Named("guest_link_id", metadata.GuestLink.ID),
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

	rowID, err := r.LastInsertId()
	if err != nil {
		log.Printf("failed to retrieve insert ID of entry %v: %v", metadata.ID, err)
		return err
	}

	_, err = s.ctx.Exec(`SELECT writeblob('main', 'entries', 'contents', :id, :offset, :data)`,
		sql.Named("id", rowID),
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

	if _, err := s.ctx.Exec(`
	DELETE FROM
		entries
	WHERE
		id = :entry_id`, sql.Named("entry_id", id)); err != nil {
		return err
	}

	return nil
}
