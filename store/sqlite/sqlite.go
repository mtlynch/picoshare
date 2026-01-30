package sqlite

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/blobio"

	"github.com/mtlynch/picoshare/picoshare"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = uint64(32768 * 10)
)

type (
	Store struct {
		ctx       *sql.DB
		chunkSize uint64
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
func NewWithChunkSize(path string, chunkSize uint64, optimizeForLitestream bool) Store {
	log.Printf("reading DB from %s", path)
	ctx, err := driver.Open(path, blobio.Register)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := ctx.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = 1;
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

func formatFileLifetime(lt picoshare.FileLifetime) string {
	return lt.String()
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func parseFileLifetime(s string) (picoshare.FileLifetime, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return picoshare.FileLifetime{}, err
	}
	return picoshare.NewFileLifetimeFromDuration(d)
}

// ClearAll deletes all data from the database (for testing only).
func (s Store) ClearAll() error {
	tables := []string{"downloads", "entries_data", "entries", "guest_links", "settings"}
	for _, table := range tables {
		if _, err := s.ctx.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}
	// Reset SQLite autoincrement sequences (if table exists)
	// Note: sqlite_sequence only exists if tables use AUTOINCREMENT
	_, _ = s.ctx.Exec("DELETE FROM sqlite_sequence")

	// Clean up /tmp to prevent disk space issues during repeated large uploads
	tmpDir, err := os.Open("/tmp")
	if err == nil {
		defer tmpDir.Close()
		entries, err := tmpDir.Readdirnames(-1)
		if err == nil {
			for _, entry := range entries {
				// Skip protected directories
				if entry == "." || entry == ".." || entry == "picoshare-data" {
					continue
				}
				_ = os.RemoveAll(filepath.Join("/tmp", entry))
			}
		}
	}

	return nil
}
