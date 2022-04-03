package sqlite

import (
	"context"
	"database/sql"
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

type (
	db struct {
		ctx       *sql.DB
		chunkSize int
	}

	dbMigration []string
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

	stmt, err := ctx.Prepare(`PRAGMA user_version`)
	if err != nil {
		log.Fatalf("failed to get user_version: %v", err)
	}
	defer stmt.Close()

	var version int
	err = stmt.QueryRow().Scan(&version)
	if err != nil {
		log.Fatalf("failed to get user_version: %v", err)
	}

	// NOTE: Migrations 0 and 1 represent the table schema as of 1.0.6. Migrations
	// 3 and 4 mutate the table to add functionality for users who have already
	// provisioned their database in <= 1.0.6.
	migrations := []dbMigration{
		// 0: Create entries table.
		{
			`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			filename TEXT,
			content_type TEXT,
			upload_time TEXT,
			expiration_time TEXT
			)`,
		},
		// 1: Create entries_data table.
		{
			`CREATE TABLE IF NOT EXISTS entries_data (
			id TEXT,
			chunk_index INTEGER,
			chunk BLOB,
			FOREIGN KEY(id) REFERENCES entries(id)
			)`,
		},
		// 3: Create guest_links table and reference it from entries table.
		{
			`CREATE TABLE IF NOT EXISTS guest_links (
				id TEXT PRIMARY KEY,
				label TEXT,
				uploads_left INTEGER,
				upload_bytes_left INTEGER,
				expiration_time TEXT
				)`,
			`ALTER TABLE entries RENAME TO old_entries`,
			`CREATE TABLE IF NOT EXISTS entries (
				id TEXT PRIMARY KEY,
				filename TEXT,
				content_type TEXT,
				upload_time TEXT,
				expiration_time TEXT,
				guest_link_id TEXT,
				FOREIGN KEY(guest_link_id) REFERENCES guest_links(id)
				)`,
			`INSERT INTO entries SELECT *, '' FROM old_entries`,
			`DROP TABLE old_entries`,
		},
		// 4: Add label column to entries table.
		{
			`ALTER TABLE entries ADD COLUMN label TEXT`,
		},
	}

	log.Printf("Migration counter: %d/%d", version, len(migrations))

	for i, migration := range migrations[version:] {
		mIdx := version + i

		tx, err := ctx.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatalf("failed to create migration transaction %d: %v", mIdx, err)
		}

		for _, stmt := range migration {
			_, err = tx.Exec(stmt)
			if err != nil {
				log.Fatalf("failed to perform DB migration %d: %v", mIdx, err)
			}
		}

		_, err = tx.Exec(fmt.Sprintf(`pragma user_version=%d`, mIdx+1))
		if err != nil {
			log.Fatalf("failed to update DB version to %d: %v", mIdx+1, err)
		}

		if err = tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration %d: %v", mIdx, err)
		}

		log.Printf("Migration counter: %d/%d", mIdx+1, len(migrations))
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
		var contentType string
		var uploadTimeRaw string
		var expirationTimeRaw string
		var fileSize int
		err = rows.Scan(&id, &filename, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSize)
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
			ContentType: types.ContentType(contentType),
			Uploaded:    ut,
			Expires:     types.ExpirationTime(et),
			Size:        fileSize,
		})
	}

	return ee, nil
}

func (d db) GetEntry(id types.EntryID) (types.UploadEntry, error) {
	stmt, err := d.ctx.Prepare(`
		SELECT
			filename,
			content_type,
			upload_time,
			expiration_time
		FROM
			entries
		WHERE
			id=?`)
	if err != nil {
		return types.UploadEntry{}, err
	}
	defer stmt.Close()

	var filename string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	err = stmt.QueryRow(id).Scan(&filename, &contentType, &uploadTimeRaw, &expirationTimeRaw)
	if err == sql.ErrNoRows {
		return types.UploadEntry{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return types.UploadEntry{}, err
	}

	ut, err := parseDatetime(uploadTimeRaw)
	if err != nil {
		return types.UploadEntry{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return types.UploadEntry{}, err
	}

	r, err := file.NewReader(d.ctx, id)
	if err != nil {
		return types.UploadEntry{}, err
	}

	return types.UploadEntry{
		UploadMetadata: types.UploadMetadata{
			ID:          id,
			Filename:    types.Filename(filename),
			ContentType: types.ContentType(contentType),
			Uploaded:    ut,
			Expires:     types.ExpirationTime(et),
		},
		Reader: r,
	}, nil
}

func (d db) InsertEntry(reader io.Reader, metadata types.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)
	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT INTO
		entries
	(
		id,
		filename,
		content_type,
		upload_time,
		expiration_time
	)
	VALUES(?,?,?,?,?)`, metadata.ID, metadata.Filename, metadata.ContentType, formatTime(metadata.Uploaded), formatTime(time.Time(metadata.Expires)))
	if err != nil {
		log.Printf("insert into entries table failed, aborting transaction: %v", err)
		return err
	}

	w := file.NewWriter(tx, metadata.ID, d.chunkSize)
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}

	// Close() flushes the buffer, and it can fail.
	if err := w.Close(); err != nil {
		return err
	}

	return tx.Commit()
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

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
