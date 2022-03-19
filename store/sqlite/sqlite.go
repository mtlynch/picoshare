package sqlite

import (
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

const timeFormat = time.RFC3339

type db struct {
	ctx *sql.DB
}

func New(path string) store.Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	initStmts := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			filename TEXT,
			content_type TEXT,
			upload_time TEXT,
			expiration_time TEXT,
			data BLOB
			)`,
	}
	for _, stmt := range initStmts {
		_, err = ctx.Exec(stmt)
		if err != nil {
			log.Fatalln(err)
		}
	}

	return &db{
		ctx: ctx,
	}
}

func (d db) GetEntriesMetadata() ([]types.UploadMetadata, error) {
	rows, err := d.ctx.Query(`
	SELECT
		id,
		filename,
		content_type,
		upload_time,
		expiration_time,
		LENGTH(data) AS file_size
	FROM
		entries`)
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
			expiration_time,
			data
		FROM
			entries
		WHERE
			id=? AND
			-- TODO: Purge expired records instead of filtering them here.
			expiration_time >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
			`)
	if err != nil {
		return types.UploadEntry{}, err
	}
	defer stmt.Close()

	var filename string
	var contentType string
	var uploadTimeRaw string
	var expirationTimeRaw string
	var data []byte
	err = stmt.QueryRow(id).Scan(&filename, &contentType, &uploadTimeRaw, &expirationTimeRaw, &data)
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

	return types.UploadEntry{
		UploadMetadata: types.UploadMetadata{
			ID:          id,
			Filename:    types.Filename(filename),
			ContentType: types.ContentType(contentType),
			Uploaded:    ut,
			Expires:     types.ExpirationTime(et),
		},
		Reader: bytes.NewReader(data),
	}, nil
}

func (d db) InsertEntry(reader io.Reader, metadata types.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("couldn't insert entry in datastore because data read failed: %v", err)
		return err
	}

	_, err = d.ctx.Exec(`
	INSERT INTO
		entries
	(
		id,
		filename,
		content_type,
		upload_time,
		expiration_time,
		data
	)
	VALUES(?,?,?,?,?,?)`, metadata.ID, metadata.Filename, metadata.ContentType, formatTime(metadata.Uploaded), formatTime(time.Time(metadata.Expires)), data)
	if err != nil {
		log.Printf("insert into DB failed: %v", err)
		return err
	}

	return nil
}

func (d db) DeleteEntry(id types.EntryID) error {
	log.Printf("deleting entry %v", id)
	_, err := d.ctx.Exec(`
	DELETE FROM
		entries
	WHERE
		id=?`, id)
	return err
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
