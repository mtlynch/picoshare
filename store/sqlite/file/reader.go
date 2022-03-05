package file

import (
	"bytes"
	"database/sql"
	"io"
	"log"

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

	startChunk := fr.offset / int64(fr.chunkSize)
	log.Printf("offset=%d, len(p)=%d, startChunk=%d)", fr.offset, len(p), startChunk)
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
	if err != nil {
		return bytesRead, err
	}
	defer rows.Close()

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

		readStart := int(fr.offset % int64(fr.chunkSize))
		readLen := min(len(p), fr.fileLength-int(fr.offset)) // TODO: Don't downcast
		readLen = min(readLen, len(chunk))
		log.Printf("readStart=%d, readLen=%d", readStart, readLen)
		copy(p[bytesRead:bytesRead+readLen], chunk[readStart:readStart+readLen])
		bytesRead += readLen
		fr.offset += int64(bytesRead)
		if bytesRead >= len(p) {
			log.Printf("read %d bytes into %d buffer, returning", bytesRead, len(p))
			break
		}
		if fr.offset == int64(fr.fileLength) {
			return bytesRead, io.EOF
		}
	}

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
