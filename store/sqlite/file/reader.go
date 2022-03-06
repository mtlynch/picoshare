package file

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"runtime"

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
	read := 0
	for {
		n, err := fr.buf.Read(p[read:])
		read += n
		log.Printf("read %d bytes from buffer, err=%v, unread=%d", n, err, fr.buf.Len())
		if err == io.EOF {
			log.Printf("fr.offset=%d, fr.fileLength=%d", fr.offset, fr.fileLength)
			if int(fr.offset) == fr.fileLength {
				return read, io.EOF
			}
			printMemUsage()
			err = fr.populateBuffer()
			printMemUsage()
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
	fr.buf = bytes.NewBuffer([]byte{})
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

func (fr *fileReader) populateBuffer() error {
	log.Printf("populating read buffer")
	if fr.offset == int64(fr.fileLength) {
		return io.EOF
	}

	startChunk := fr.offset / int64(fr.chunkSize)
	log.Printf("offset=%d, startChunk=%d", fr.offset, startChunk)
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
		return err
	}
	defer stmt.Close()

	var chunk []byte
	err = stmt.QueryRow(fr.entryID, startChunk).Scan(&chunk)
	if err != nil {
		return err
	}

	// Move the start index to the position in the chunk we want to read.
	readStart := fr.offset % int64(fr.chunkSize)

	fr.buf = bytes.NewBuffer(chunk[readStart:])
	fr.offset += int64(len(chunk)) - readStart

	return nil
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

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("alloc: %v MiB, Sys: %v MiB, tNumGC: %v", bToMb(m.Alloc), bToMb(m.Sys), m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
