package sqlite

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

const timeFormat = time.RFC3339

type db struct {
	ctx *sql.DB
}

func New() store.Store {
	dbDir := "data"
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		os.Mkdir(dbDir, os.ModePerm)
	}
	ctx, err := sql.Open("sqlite3", dbDir+"/store.db")
	if err != nil {
		log.Fatalln(err)
	}

	initStmts := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			filename TEXT,
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

func (d db) GetEntry(id types.EntryID) (types.UploadEntry, error) {
	stmt, err := d.ctx.Prepare(`
		SELECT
			filename,
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
	var uploadTimeRaw string
	var expirationTimeRaw string
	var data []byte
	err = stmt.QueryRow(id).Scan(&filename, &uploadTimeRaw, &expirationTimeRaw, &data)
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
		Filename: types.Filename(filename),
		Uploaded: ut,
		Expires:  et,
		Data:     data,
	}, nil
}

func (d db) InsertEntry(id types.EntryID, entry types.UploadEntry) error {
	log.Printf("saving new entry %s (%d bytes)", id, len(entry.Data))
	_, err := d.ctx.Exec(`
	INSERT INTO
		entries
	(
		id,
		filename,
		upload_time,
		expiration_time,
		data
	)
	VALUES(?,?,?,?,?)`, id, entry.Filename, formatTime(entry.Uploaded), formatTime(entry.Expires), entry.Data)
	return err
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
