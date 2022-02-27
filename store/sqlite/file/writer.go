package file

import (
	"database/sql"
	"errors"
	"io"

	"github.com/mtlynch/picoshare/v2/types"
)

type (
	sqlTx interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
	}

	writer struct {
		tx      sqlTx
		entryID types.EntryID
		buf     []byte
		written int
	}
)

func NewWriter(tx sqlTx, id types.EntryID, chunkSize int) io.WriteCloser {
	return &writer{
		tx:      tx,
		entryID: id,
		buf:     make([]byte, chunkSize),
	}
}

func (w *writer) Write(p []byte) (int, error) {
	n := 0

	dstStart := w.written % len(buf)

	if len(p) < len(w.buf) {
		copy(w.buf, p)
		w.written += len(p)
		return len(p), w.flush()
	}
	// TODO: Handle when p is smaller

	return n, nil
}

func (w *writer) Close() error {
	return errors.New("not implemented")
}

func (w *writer) flush() error {
	idx := w.written / ChunkSize
	n := w.written % ChunkSize
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
