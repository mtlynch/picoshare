package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"math"
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

func formatFileExpirationTime(et picoshare.ExpirationTime) string {
	delta := time.Until(time.Time(et))
	return fmt.Sprintf("%.0f", math.Abs(delta.Hours())/24)
}

func formatExpirationTime(et picoshare.ExpirationTime) string {
	return formatTime(time.Time(et))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseFileDatetime(s string) time.Time {
	t := time.Now()
	switch s {
	case "1":
		t = t.AddDate(0, 0, 1)
	case "7":
		t = t.AddDate(0, 0, 7)
	case "30":
		t = t.AddDate(0, 0, 30)
	case "365":
		t = t.AddDate(1, 0, 0)
	default:
		t = time.Time(picoshare.NeverExpire)
	}

	return t
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
