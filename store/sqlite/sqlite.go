package sqlite

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = 32768 * 10
)

type (
	Store struct {
		ctx       *sql.DB
		chunkSize int
	}

	rowScanner interface {
		Scan(...interface{}) error
	}
)

func New(path string, optimizeForLitestream bool) Store {
	return NewWithChunkSize(path, defaultChunkSize, optimizeForLitestream)
}

// NewWithChunkSize creates a SQLite-based datastore with the user-specified
// chunk size for writing files. Most callers should just use New().
func NewWithChunkSize(path string, chunkSize int, optimizeForLitestream bool) Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
		`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	if optimizeForLitestream {
		if _, err := ctx.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
				`); err != nil {
			log.Fatalf("failed to set Litestream compatibility pragmas: %v", err)
		}
	}

	applyMigrations(ctx)

	return Store{
		ctx:       ctx,
		chunkSize: chunkSize,
	}
}

func formatExpirationTime(et picoshare.ExpirationTime) string {
	return formatTime(time.Time(et))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
