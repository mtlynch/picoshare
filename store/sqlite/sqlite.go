package sqlite

import (
	"context"
	"database/sql"
	"io"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = 32768 * 10
)

type (
	db struct {
		ctx       *sql.DB
		chunkSize int
	}

	rowScanner interface {
		Scan(...interface{}) error
	}
)

func New(path string, optimizeForLitestream bool) store.Store {
	return NewWithChunkSize(path, defaultChunkSize, optimizeForLitestream)
}

// NewWithChunkSize creates a SQLite-based datastore with the user-specified
// chunk size for writing files. Most callers should just use New().
func NewWithChunkSize(path string, chunkSize int, optimizeForLitestream bool) store.Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
		`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	if optimizeForLitestream {
		if _, err := ctx.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
				`); err != nil {
			log.Fatalf("failed to set Litestream compatibility pragmas: %v", err)
		}
	}

	applyMigrations(ctx)

	return &db{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}

func (d db) GetEntriesMetadata() ([]picoshare.UploadMetadata, error) {
	rows, err := d.ctx.Query(`
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
		var fileSize int64
		err = rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSize)
		if err != nil {
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
			ID:          picoshare.EntryID(id),
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

func (d db) GetEntry(id picoshare.EntryID) (picoshare.UploadEntry, error) {
	metadata, err := d.GetEntryMetadata(id)
	if err != nil {
		return picoshare.UploadEntry{}, err
	}

	r, err := file.NewReader(d.ctx, id)
	if err != nil {
		return picoshare.UploadEntry{}, err
	}

	return picoshare.UploadEntry{
		UploadMetadata: metadata,
		Reader:         r,
	}, nil
}

func (d db) GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	var filename string
	var note *string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	err := d.ctx.QueryRow(`
	SELECT
		filename,
		note,
		content_type,
		upload_time,
		expiration_time
	FROM
		entries
	WHERE
		id=?`, id).Scan(&filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw)
	if err == sql.ErrNoRows {
		return picoshare.UploadMetadata{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.UploadMetadata{}, err
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
		Note:        picoshare.FileNote{Value: note},
		ContentType: picoshare.ContentType(contentType),
		Uploaded:    ut,
		Expires:     picoshare.ExpirationTime(et),
	}, nil
}

func (d db) InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	// Note: We deliberately don't use a transaction here, as it bloats memory, so
	// we can end up in a state with orphaned entries data. We clean it up in
	// Purge().
	// See: https://github.com/mtlynch/picoshare/issues/284

	w := file.NewWriter(d.ctx, metadata.ID, d.chunkSize)
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	// Close() flushes the buffer, and it can fail.
	if err := w.Close(); err != nil {
		return err
	}

	_, err := d.ctx.Exec(`
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
	VALUES(?,?,?,?,?,?,?)`,
		metadata.ID,
		metadata.GuestLinkID,
		metadata.Filename,
		metadata.Note.Value,
		metadata.ContentType,
		formatTime(metadata.Uploaded),
		formatExpirationTime(metadata.Expires),
	)
	if err != nil {
		log.Printf("insert into entries table failed, aborting transaction: %v", err)
		return err
	}

	return nil
}

func (d db) UpdateEntryMetadata(id picoshare.EntryID, metadata picoshare.UploadMetadata) error {
	log.Printf("updating metadata for entry %s", id)

	res, err := d.ctx.Exec(`
	UPDATE entries
	SET
		filename = ?,
		expiration_time = ?,
		note = ?
	WHERE
		id=?`,
		metadata.Filename,
		formatExpirationTime(metadata.Expires),
		metadata.Note.Value,
		id)
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

func (d db) DeleteEntry(id picoshare.EntryID) error {
	log.Printf("deleting entry %v", id)

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries
	WHERE
		id=?`, id)
	if err != nil {
		log.Printf("delete from entries table failed, aborting transaction: %v", err)
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id=?`, id)
	if err != nil {
		log.Printf("delete from entries_data table failed, aborting transaction: %v", err)
		return err
	}

	return tx.Commit()
}

func (d db) GetGuestLink(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
	row := d.ctx.QueryRow(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.expiration_time AS expiration_time,
			SUM(CASE WHEN entries.id IS NOT NULL THEN 1 ELSE 0 END) AS entry_count
		FROM
			guest_links
		LEFT JOIN
			entries ON guest_links.id = entries.guest_link_id
		WHERE
			guest_links.id=?
		GROUP BY
			guest_links.id`, id)

	return guestLinkFromRow(row)
}

func (d db) GetGuestLinks() ([]picoshare.GuestLink, error) {
	rows, err := d.ctx.Query(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.expiration_time AS expiration_time,
			SUM(CASE WHEN entries.id IS NOT NULL THEN 1 ELSE 0 END) AS entry_count
		FROM
			guest_links
		LEFT JOIN
			entries ON guest_links.id = entries.guest_link_id
		GROUP BY
			guest_links.id`)
	if err != nil {
		return []picoshare.GuestLink{}, err
	}

	gls := []picoshare.GuestLink{}
	for rows.Next() {
		gl, err := guestLinkFromRow(rows)
		if err != nil {
			return []picoshare.GuestLink{}, err
		}

		gls = append(gls, gl)
	}

	return gls, nil
}

func (d *db) InsertGuestLink(guestLink picoshare.GuestLink) error {
	log.Printf("saving new guest link %s", guestLink.ID)

	if _, err := d.ctx.Exec(`
	INSERT INTO guest_links
		(
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			expiration_time
		)
		VALUES (?,?,?,?,?,?)
	`, guestLink.ID, guestLink.Label, guestLink.MaxFileBytes, guestLink.MaxFileUploads, formatTime(time.Now()), formatExpirationTime(guestLink.Expires)); err != nil {
		return err
	}

	return nil
}

func (d db) DeleteGuestLink(id picoshare.GuestLinkID) error {
	log.Printf("deleting guest link %s", id)

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		guest_links
	WHERE
		id=?`, id)
	if err != nil {
		log.Printf("deleting %s from guest_links table failed: %v", id, err)
		return err
	}

	_, err = tx.Exec(`
	UPDATE
		entries
	SET
		guest_link_id = NULL
	WHERE
		guest_link_id = ?`, id)
	if err != nil {
		log.Printf("removing references to guest link %s from entries table failed: %v", id, err)
		return err
	}

	return tx.Commit()
}

// Purge deletes expired entries and clears orphaned rows from the database.
func (d db) Purge() error {
	if err := d.deleteExpiredEntries(); err != nil {
		return err
	}

	if err := d.deleteOrphanedRows(); err != nil {
		return err
	}

	return nil
}

func (d db) Compact() error {
	log.Printf("vacuuming database")

	if _, err := d.ctx.Exec("VACUUM"); err != nil {
		log.Printf("failed to vacuum database: %v", err)
		return err
	}

	log.Printf("vacuuming complete")

	return nil
}

func (d db) deleteExpiredEntries() error {
	log.Printf("deleting expired entries from database")

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	currentTime := formatTime(time.Now())

	_, err = tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id IN (
			SELECT
				id
			FROM
				entries
			WHERE
				entries.expiration_time IS NOT NULL AND
				entries.expiration_time < ?
		);`, currentTime)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries
	WHERE
		entries.expiration_time IS NOT NULL AND
		entries.expiration_time < ?;
	`, currentTime)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d db) deleteOrphanedRows() error {
	log.Printf("purging orphaned rows from database")

	// Delete rows from entries_data if they don't reference valid rows in
	// entries. This can happen if the entry insertion fails partway through.
	if _, err := d.ctx.Exec(`
		DELETE FROM
			entries_data
		WHERE
		id IN (
			SELECT
				DISTINCT entries_data.id AS entry_id
			FROM
				entries_data
			LEFT JOIN
				entries ON entries_data.id = entries.id
			WHERE
				entries.id IS NULL
			)`); err != nil {
		return err
	}

	log.Printf("purge completed successfully")

	return nil
}

func guestLinkFromRow(row rowScanner) (picoshare.GuestLink, error) {
	var id picoshare.GuestLinkID
	var label picoshare.GuestLinkLabel
	var maxFileBytes picoshare.GuestUploadMaxFileBytes
	var maxFileUploads picoshare.GuestUploadCountLimit
	var creationTimeRaw string
	var expirationTimeRaw string
	var filesUploaded int

	err := row.Scan(&id, &label, &maxFileBytes, &maxFileUploads, &creationTimeRaw, &expirationTimeRaw, &filesUploaded)
	if err == sql.ErrNoRows {
		return picoshare.GuestLink{}, store.GuestLinkNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.GuestLink{}, err
	}

	ct, err := parseDatetime(creationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	return picoshare.GuestLink{
		ID:             id,
		Label:          label,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
		FilesUploaded:  filesUploaded,
		Created:        ct,
		Expires:        picoshare.ExpirationTime(et),
	}, nil
}

func formatExpirationTime(et picoshare.ExpirationTime) string {
	return formatTime(time.Time(et))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
