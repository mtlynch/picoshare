package file

import (
	"io"
	"log"

	"github.com/mtlynch/picoshare/v2/types"
)

type writer struct {
	db      SqlDB
	entryID types.EntryID
	buf     []byte
	written int
}

func NewWriter(db SqlDB, id types.EntryID, chunkSize int) io.WriteCloser {
	return &writer{
		db:      db,
		entryID: id,
		buf:     make([]byte, chunkSize),
	}
}

func (w *writer) Write(p []byte) (int, error) {
	n := 0

	for {
		if n == len(p) {
			break
		}
		dstStart := w.written % len(w.buf)
		copySize := min(len(w.buf)-dstStart, len(p)-n)
		dstEnd := dstStart + copySize
		log.Printf("dstStart=%d, dstEnd=%d, n=%d, copySize=%d", dstStart, dstEnd, n, copySize)
		copy(w.buf[dstStart:dstEnd], p[n:n+copySize])
		if dstEnd == len(w.buf) {
			w.flush(len(w.buf))
		}
		w.written += copySize
		n += copySize
	}

	return n, nil
}

func (w *writer) Close() error {
	unflushed := w.written % len(w.buf)
	if unflushed != 0 {
		return w.flush(unflushed)
	}
	return nil
}

func (w *writer) flush(n int) error {
	idx := w.written / len(w.buf)
	log.Printf("flushing %s -> idx=%d, n=%d", w.entryID, idx, n)

	// HACK: Try to shrink memory before inserting more data.
	_, err := w.db.Exec(`PRAGMA shrink_memory`)
	if err != nil {
		return err
	}
	_, err = w.db.Exec(`
	INSERT INTO
		entries_data
	(
		id,
		chunk_index,
		chunk
	)
	VALUES(?,?,?)`, w.entryID, idx, w.buf[0:n])

	return err
}
