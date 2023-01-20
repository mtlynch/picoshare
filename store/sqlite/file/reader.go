package file

import (
	"bytes"
	"database/sql"
	"io"
	"log"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

type (
	fileReader struct {
		db         *sql.DB
		entryID    picoshare.EntryID
		fileLength int64
		offset     int64
		chunkSize  int64
		buf        *bytes.Buffer
	}
)

func NewReader(db *sql.DB, id picoshare.EntryID) (io.ReadSeeker, error) {
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

func (fr *fileReader) Read(p []byte) (int, error) {
	read := 0
	for {
		n, err := fr.buf.Read(p[read:])
		read += n
		if err == io.EOF {
			// If we've reached the end of the buffer, then we've read the full file.
			if fr.offset == fr.fileLength {
				return read, io.EOF
			}
			// Otherwise, repopulate the buffer with the underlying SQLite DB and
			// continue reading.
			err = fr.populateBuffer()
			if err != nil {
				return read, err
			}
			continue
		}
		if read >= len(p) {
			break
		}
	}

	return read, nil
}

func (fr *fileReader) Seek(offset int64, whence int) (int64, error) {
	// Seeking to a new position invalidates the buffer, so reset to zero.
	fr.buf = bytes.NewBuffer([]byte{})

	switch whence {
	case io.SeekStart:
		fr.offset = offset
	case io.SeekCurrent:
		fr.offset += offset
	case io.SeekEnd:
		fr.offset = int64(fr.fileLength) - offset
	}
	return fr.offset, nil
}

func (fr *fileReader) populateBuffer() error {
	if fr.offset == int64(fr.fileLength) {
		return io.EOF
	}

	chunkIndex := fr.offset / int64(fr.chunkSize)
	var chunk []byte
	if err := fr.db.QueryRow(`
			SELECT
				chunk
			FROM
				entries_data
			WHERE
				id=? AND
				chunk_index=?
			ORDER BY
				chunk_index ASC
			`, fr.entryID, chunkIndex).Scan(&chunk); err != nil {
		log.Printf("reading chunk failed: %v", err)
		return err
	}

	// Move the start index to the position in the chunk we want to read.
	readStart := fr.offset % int64(fr.chunkSize)

	fr.buf = bytes.NewBuffer(chunk[readStart:])
	fr.offset += int64(len(chunk)) - readStart

	return nil
}

func getFileLength(db *sql.DB, id picoshare.EntryID, chunkSize int64) (int64, error) {
	var chunkIndex int64
	var chunkLen int64
	if err := db.QueryRow(`
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
	`, id).Scan(&chunkIndex, &chunkLen); err != nil {
		return 0, err
	}

	return (chunkSize * chunkIndex) + chunkLen, nil
}

func getChunkSize(db *sql.DB, id picoshare.EntryID) (int64, error) {
	var chunkSize int64
	if err := db.QueryRow(`
	SELECT
		LENGTH(chunk) AS chunk_size
	FROM
		entries_data
	WHERE
		id=?
	ORDER BY
		chunk_index ASC
	LIMIT 1
	`, id).Scan(&chunkSize); err != nil {
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
