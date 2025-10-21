package file

import (
	"io"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/sqlite/wrapped"
)

type writer struct {
	ctx     wrapped.SqlDB
	entryID picoshare.EntryID
	buf     []byte
	written int
}

// Create a new writer for the entry ID using the given SqlTx and splitting the
// file into separate rows in the DB of at most chunkSize bytes.
func NewWriter(ctx wrapped.SqlDB, id picoshare.EntryID, chunkSize uint64) io.WriteCloser {
	return &writer{
		ctx:     ctx,
		entryID: id,
		buf:     make([]byte, chunkSize),
	}
}

// Write writes a buffer to the SQLite database.
func (w *writer) Write(p []byte) (int, error) {
	n := 0

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

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
	_, err := w.ctx.Exec(`
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
