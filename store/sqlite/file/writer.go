package file

import (
	"io"

	"github.com/mtlynch/picoshare/v2/store/sqlite/wrapped"
	"github.com/mtlynch/picoshare/v2/types"
)

type writer struct {
	tx      wrapped.SqlTx
	entryID types.EntryID
	buf     []byte
	written int
}

// Create a new writer for the entry ID using the given SqlTx and splitting the
// file into separate rows in the DB of at most chunkSize bytes.
func NewWriter(tx wrapped.SqlTx, id types.EntryID, chunkSize int) io.WriteCloser {
	return &writer{
		tx:      tx,
		entryID: id,
		buf:     make([]byte, chunkSize),
	}
}

// Write writes a buffer to the SQLite database.
func (w *writer) Write(p []byte) (int, error) {
	n := 0

	for {
		if n == len(p) {
			break
		}
		dstStart := w.written % len(w.buf)
		copySize := min(len(w.buf)-dstStart, len(p)-n)
		dstEnd := dstStart + copySize
		copy(w.buf[dstStart:dstEnd], p[n:n+copySize])
		if dstEnd == len(w.buf) {
			if err := w.flush(len(w.buf)); err != nil {
				return n, err
			}
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
	_, err := w.tx.Exec(`
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
