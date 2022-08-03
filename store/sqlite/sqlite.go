package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
	"github.com/mtlynch/picoshare/v2/types"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = 32768 * 10
)

//go:embed migrations/*.sql
var migrationsFs embed.FS

type (
	db struct {
		ctx       *sql.DB
		chunkSize int
	}

	dbMigration struct {
		version int
		query   string
	}

	rowScanner interface {
		Scan(...interface{}) error
	}
)

func New(path string) store.Store {
	return NewWithChunkSize(path, defaultChunkSize)
}

// NewWithChunkSize creates a SQLite-based datastore with the user-specified
// chunk size for writing files. Most callers should just use New().
func NewWithChunkSize(path string, chunkSize int) store.Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
PRAGMA temp_store = FILE;

-- Apply Litestream recommendations: https://litestream.io/tips/
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA journal_mode = WAL;
PRAGMA wal_autocheckpoint = 0;
		`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	var version int
	if err := ctx.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		log.Fatalf("failed to get user_version: %v", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		log.Fatalf("error loading database migrations: %v", err)
	}

	log.Printf("Migration counter: %d/%d", version, len(migrations))

	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		tx, err := ctx.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatalf("failed to create migration transaction %d: %v", migration.version, err)
		}

		_, err = tx.Exec(migration.query)
		if err != nil {
			log.Fatalf("failed to perform DB migration %d: %v", migration.version, err)
		}

		_, err = tx.Exec(fmt.Sprintf(`pragma user_version=%d`, migration.version))
		if err != nil {
			log.Fatalf("failed to update DB version to %d: %v", migration.version, err)
		}

		if err = tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration %d: %v", migration.version, err)
		}

		log.Printf("Migration counter: %d/%d", migration.version, len(migrations))
	}

	return &db{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}

func (d db) GetEntriesMetadata() ([]types.UploadMetadata, error) {
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
		return []types.UploadMetadata{}, err
	}

	ee := []types.UploadMetadata{}
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
			return []types.UploadMetadata{}, err
		}

		ut, err := parseDatetime(uploadTimeRaw)
		if err != nil {
			return []types.UploadMetadata{}, err
		}

		et, err := parseDatetime(expirationTimeRaw)
		if err != nil {
			return []types.UploadMetadata{}, err
		}

		ee = append(ee, types.UploadMetadata{
			ID:          types.EntryID(id),
			Filename:    types.Filename(filename),
			Note:        types.FileNote{Value: note},
			ContentType: types.ContentType(contentType),
			Uploaded:    ut,
			Expires:     types.ExpirationTime(et),
			Size:        fileSize,
		})
	}

	return ee, nil
}

func (d db) GetEntry(id types.EntryID) (types.UploadEntry, error) {
	metadata, err := d.GetEntryMetadata(id)
	if err != nil {
		return types.UploadEntry{}, err
	}

	r, err := file.NewReader(d.ctx, id)
	if err != nil {
		return types.UploadEntry{}, err
	}

	return types.UploadEntry{
		UploadMetadata: metadata,
		Reader:         r,
	}, nil
}

func (d db) GetEntryMetadata(id types.EntryID) (types.UploadMetadata, error) {
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
		return types.UploadMetadata{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return types.UploadMetadata{}, err
	}

	ut, err := parseDatetime(uploadTimeRaw)
	if err != nil {
		return types.UploadMetadata{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return types.UploadMetadata{}, err
	}

	return types.UploadMetadata{
		ID:          id,
		Filename:    types.Filename(filename),
		Note:        types.FileNote{Value: note},
		ContentType: types.ContentType(contentType),
		Uploaded:    ut,
		Expires:     types.ExpirationTime(et),
	}, nil
}

func (d db) InsertEntry(reader io.Reader, metadata types.UploadMetadata) error {
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

func (d db) UpdateEntryMetadata(id types.EntryID, metadata types.UploadMetadata) error {
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

func (d db) DeleteEntry(id types.EntryID) error {
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

func (d db) GetGuestLink(id types.GuestLinkID) (types.GuestLink, error) {
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

func (d db) GetGuestLinks() ([]types.GuestLink, error) {
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
		return []types.GuestLink{}, err
	}

	gls := []types.GuestLink{}
	for rows.Next() {
		gl, err := guestLinkFromRow(rows)
		if err != nil {
			return []types.GuestLink{}, err
		}

		gls = append(gls, gl)
	}

	return gls, nil
}

func (d *db) InsertGuestLink(guestLink types.GuestLink) error {
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

func (d db) DeleteGuestLink(id types.GuestLinkID) error {
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

// Purge clears orphaned rows from the database.
func (d db) Purge() error {
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

func (d db) Compact() error {
	log.Printf("vacuuming database")

	if _, err := d.ctx.Exec("VACUUM"); err != nil {
		log.Printf("failed to vacuum database: %v", err)
		return err
	}

	log.Printf("vacuuming complete")

	return nil
}

func guestLinkFromRow(row rowScanner) (types.GuestLink, error) {
	var id types.GuestLinkID
	var label types.GuestLinkLabel
	var maxFileBytes types.GuestUploadMaxFileBytes
	var maxFileUploads types.GuestUploadCountLimit
	var creationTimeRaw string
	var expirationTimeRaw string
	var filesUploaded int

	err := row.Scan(&id, &label, &maxFileBytes, &maxFileUploads, &creationTimeRaw, &expirationTimeRaw, &filesUploaded)
	if err == sql.ErrNoRows {
		return types.GuestLink{}, store.GuestLinkNotFoundError{ID: id}
	} else if err != nil {
		return types.GuestLink{}, err
	}

	ct, err := parseDatetime(creationTimeRaw)
	if err != nil {
		return types.GuestLink{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return types.GuestLink{}, err
	}

	return types.GuestLink{
		ID:             id,
		Label:          label,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
		FilesUploaded:  filesUploaded,
		Created:        ct,
		Expires:        types.ExpirationTime(et),
	}, nil
}

func formatExpirationTime(et types.ExpirationTime) string {
	return formatTime(time.Time(et))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
