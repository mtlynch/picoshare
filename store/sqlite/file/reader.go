package file

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"strings"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

type (
	fileReader struct {
		db         *sql.DB
		entryID    types.EntryID
		fileLength int
		offset     int64
		chunkSize  int
		buf        *bytes.Buffer
	}
)

func NewReader(db *sql.DB, id types.EntryID) (io.ReadSeeker, error) {
	log.Printf("creating file reader")

	chunkSize, err := getChunkSize(db, id)
	if err != nil {
		return nil, err
	}

	log.Printf("getting file length")
	length, err := getFileLength(db, id, chunkSize)
	if err != nil {
		return nil, err
	}

	return &fileReader{
		db:         db,
		entryID:    id,
		fileLength: length,
		offset:     0,
		chunkSize:  chunkSize,
		buf:        bytes.NewBuffer([]byte{}),
	}, nil
}

func (fr *fileReader) Read(p []byte) (int, error) {
	if fr.offset == int64(fr.fileLength) {
		return 0, io.EOF
	}
	// TODO: Keep a buffer between reads to minimize reads to SQLite

	bytesRead := 0
	bytesToRead := min(len(p), fr.fileLength-int(fr.offset)) // TODO: Don't downcast
	startChunk := fr.offset / int64(fr.chunkSize)
	log.Printf("reading %d bytes, offset=%d, len(p)=%d, startChunk=%d)", bytesToRead, fr.offset, len(p), startChunk)
	stmt, err := fr.db.Prepare(`
			SELECT
				chunk
			FROM
				entries_data
			WHERE
				id=? AND
				chunk_index>=?
			ORDER BY
				chunk_index ASC
			`)
	if err != nil {
		log.Printf("reading chunk failed: %v", err)
		return 0, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(fr.entryID, startChunk)
	for rows.Next() {
		log.Printf("reading chunk, bytesRead=%d", bytesRead)

		var chunk []byte
		rows.Scan(&chunk)
		if err == sql.ErrNoRows {
			log.Printf("no rows!")
			// TODO: Better error
			return bytesRead, store.EntryNotFoundError{ID: fr.entryID}
		} else if err != nil {
			log.Printf("error reading chunk: %v", err)
			return bytesRead, err
		}
		fr.buf = bytes.NewBuffer(chunk)

		chunkStart := int(fr.offset % int64(fr.chunkSize))
		chunkEnd := min(len(chunk), bytesToRead-chunkStart)
		log.Printf("chunkStart=%d, chunkEnd=%d", chunkStart, chunkEnd)
		bytesToCopy := min(bytesToRead, chunkEnd-chunkStart)
		copy(p[bytesRead:bytesRead+bytesToCopy], chunk[chunkStart:bytesToCopy])
		bytesRead += bytesToCopy
		fr.offset += int64(bytesRead)
		if bytesRead >= len(p) {
			log.Printf("read %d bytes into %d buffer, returning", bytesRead, len(p))
			break
		}
		if fr.offset == int64(fr.fileLength) {
			printTables(fr.db, "within loop")
			return bytesRead, io.EOF
		}
	}

	printTables(fr.db, "before reader exit")

	return bytesRead, nil
}

func (fr *fileReader) Seek(offset int64, whence int) (int64, error) {
	log.Printf("seeking to %d, %d", offset, whence)
	log.Printf("current offset=%d", fr.offset)
	switch whence {
	case io.SeekStart:
		fr.offset = offset
	case io.SeekCurrent:
		fr.offset += offset
	case io.SeekEnd:
		fr.offset = int64(fr.fileLength) - offset
	}
	log.Printf("    new offset=%d", fr.offset)
	return fr.offset, nil
}

func getFileLength(db *sql.DB, id types.EntryID, chunkSize int) (int, error) {
	stmt, err := db.Prepare(`
	SELECT
		chunk_index,
		LENGTH(chunk) AS chunk_size
	FROM
		entries_data
	WHERE
		id=?
	ORDER BY
		chunk_index DESC
	LIMIT 1
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var chunkIndex int
	var chunkLen int64
	err = stmt.QueryRow(id).Scan(&chunkIndex, &chunkLen)
	if err != nil {
		return 0, err
	}

	return chunkSize*chunkIndex + int(chunkLen), nil
}

func getChunkSize(db *sql.DB, id types.EntryID) (int, error) {
	stmt, err := db.Prepare(`
	SELECT
		LENGTH(chunk) AS chunk_size
	FROM
		entries_data
	WHERE
		id=?
	ORDER BY
		chunk_index ASC
	LIMIT 1
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var chunkSize int
	err = stmt.QueryRow(id).Scan(&chunkSize)
	if err != nil {
		return 0, err
	}
	return chunkSize, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printTables(db *sql.DB, label string) { // DEBUG
	log.Printf("printing tables from reader: %s", label)
	tables := []string{}
	rows, err := db.Query("SELECT tbl_name FROM sqlite_master where type='table'")
	if err != nil {
		log.Printf("failed to get tables: %v", err)
	}

	for rows.Next() {
		var tblName string
		err = rows.Scan(&tblName)
		if err != nil {
			log.Printf("failed to get row: %v", err)
		}
		tables = append(tables, tblName)
	}
	log.Printf("tables(%d): %v", len(tables), strings.Join(tables, ", "))
}
