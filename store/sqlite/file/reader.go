package file

import (
	"database/sql"
	"io"
	"log"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

const ChunkSize = 32 << 20 // TODO: Get rid of this

type (
	SqlDB interface {
		Prepare(string) (*sql.Stmt, error)
	}

	fileReader struct {
		db         SqlDB
		entryID    types.EntryID
		fileLength int
		offset     int64
	}
)

func NewReader(db SqlDB, id types.EntryID) (io.ReadSeeker, error) {
	log.Printf("creating file reader")

	length, err := getFileLength(db, id)
	if err != nil {
		return nil, err
	}

	return &fileReader{
		db:         db,
		entryID:    id,
		fileLength: length,
		offset:     0,
	}, nil
}

func (cr *fileReader) Read(p []byte) (int, error) {
	log.Printf("reading %d bytes (offset=%d)", len(p), cr.offset)
	bytesRead := 0
	bytesToRead := len(p)
	startChunk := cr.offset / ChunkSize
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

	var chunk []byte
	rows, err := stmt.Query(cr.entryID, startChunk)
	for rows.Next() {
		log.Printf("reading chunk, bytesRead=%d", bytesRead)
		rows.Scan(&chunk)
		if err == sql.ErrNoRows {
			log.Printf("no rows!")
			// TODO: Better error
			return bytesRead, store.EntryNotFoundError{ID: cr.entryID}
		} else if err != nil {
			log.Printf("error reading chunk: %v", err)
			return bytesRead, err
		}
		chunkStart := int(cr.offset % ChunkSize)
		chunkEnd := min(len(chunk), bytesToRead-chunkStart)
		log.Printf("chunkStart=%d, chunkEnd=%d", chunkStart, chunkEnd)
		bytesToRead = min(bytesToRead, chunkEnd-chunkStart)
		copy(p[bytesRead:bytesRead+bytesToRead], chunk[chunkStart:bytesToRead])
		bytesRead += int(chunkEnd) - int(chunkStart)
		if bytesRead >= len(p) {
			log.Printf("read %d bytes into %d buffer, returning", bytesRead, len(p))
			break
		}
	}
	cr.offset += int64(bytesRead)

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

func getFileLength(db SqlDB, id types.EntryID) (int, error) {
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

	return ChunkSize*chunkIndex + int(chunkLen), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
