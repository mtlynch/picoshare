package file

import (
	"bytes"
	"database/sql"
	"io"
	"log"

	"github.com/mtlynch/picoshare/v2/types"
)

type (
	SqlDB interface {
		Prepare(string) (*sql.Stmt, error)
		Exec(string, ...interface{}) (sql.Result, error)
	}

	fileReader struct {
		db         SqlDB
		entryID    types.EntryID
		fileLength int
		offset     int64
		chunkSize  int
		buf        *bytes.Buffer
	}
)

func NewReader(db SqlDB, id types.EntryID) (io.ReadSeeker, error) {
	log.Printf("creating file reader")

	chunkSize, err := getChunkSize(db, id)
	if err != nil {
		return nil, err
	}

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

func (cr *fileReader) Read(p []byte) (int, error) {
	log.Printf("reading %d bytes (offset=%d)", len(p), cr.offset)
	bytesRead := 0
	bytesToRead := min(len(p), cr.fileLength-int(cr.offset)) // TODO: Don't downcast
	if bytesToRead == 0 {
		return 0, io.EOF
	}
	chunk := make([]byte, bytesToRead)
	bytesRead, err := cr.buf.Read(chunk)
	// Three cases:
	// 1 - Buffer is empty
	// 2 - Buffer is full enough to fulfill request
	// 3 - Can fulfill some of the request from buffer, but have to read again
	copy(p[0:bytesRead], chunk[0:bytesRead])
	if err == io.EOF {
		// handle EOF
	} else if err != nil {
		return bytesRead, err
	}
	// Satisfied the whole read from buffer, return
	return bytesRead, nil

	/*
		startChunk := cr.offset / int64(cr.chunkSize)
		log.Printf("startChunk=%d, bytesToRead=%d", startChunk, bytesToRead)
		stmt, err := cr.db.Prepare(`
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

		rows, err := stmt.Query(cr.entryID, startChunk)
		for rows.Next() {
			log.Printf("reading chunk, bytesRead=%d", bytesRead)

			var chunk []byte
			rows.Scan(&chunk)
			if err == sql.ErrNoRows {
				log.Printf("no rows!")
				// TODO: Better error
				return bytesRead, store.EntryNotFoundError{ID: cr.entryID}
			} else if err != nil {
				log.Printf("error reading chunk: %v", err)
				return bytesRead, err
			}
			cr.buf = bytes.NewBuffer(chunk)

			chunkStart := int(cr.offset % int64(cr.chunkSize))
			chunkEnd := min(len(chunk), bytesToRead-chunkStart)
			log.Printf("chunkStart=%d, chunkEnd=%d", chunkStart, chunkEnd)
			bytesToRead = min(bytesToRead, chunkEnd-chunkStart)
			copy(p[bytesRead:bytesRead+bytesToRead], chunk[chunkStart:bytesToRead])
			bytesRead += chunkEnd - chunkStart
			cr.offset += int64(bytesRead)
			if bytesRead >= len(p) {
				log.Printf("read %d bytes into %d buffer, returning", bytesRead, len(p))
				break
			}
			if cr.offset == int64(cr.fileLength) {
				return bytesRead, io.EOF
			}
		}*/

	return bytesRead, nil
}

func (cr *fileReader) Seek(offset int64, whence int) (int64, error) {
	log.Printf("seeking to %d, %d", offset, whence)
	log.Printf("current offset=%d", cr.offset)
	switch whence {
	case io.SeekStart:
		cr.offset = offset
	case io.SeekCurrent:
		cr.offset += offset
	case io.SeekEnd:
		cr.offset = int64(cr.fileLength) - offset
	}
	log.Printf("    new offset=%d", cr.offset)
	return cr.offset, nil
}

func getFileLength(db SqlDB, id types.EntryID, chunkSize int) (int, error) {
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

func getChunkSize(db SqlDB, id types.EntryID) (int, error) {
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
