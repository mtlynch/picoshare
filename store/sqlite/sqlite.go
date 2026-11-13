package sqlite

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/picoshare"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = uint64(32768 * 10)
)

type (
	Params struct {
		Path                  string
		OptimizeForLitestream bool
		Now                   func() time.Time
		PicoShareChunkSize    uint64
	}

	Store struct {
		db        *sql.DB
		chunkSize uint64
		now       func() time.Time
	}

	rowScanner interface {
		Scan(...any) error
	}
)

func New(params Params) Store {
	log.Printf("reading DB from %s", params.Path)
	db, err := sql.Open("sqlite3", params.Path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := db.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = 1;
		`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	if params.OptimizeForLitestream {
		if _, err := db.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
				`); err != nil {
			log.Fatalf("failed to set Litestream compatibility pragmas: %v", err)
		}
	}

	applyMigrations(db)

	chunkSize := params.PicoShareChunkSize
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
	}

	return Store{
		db:        db,
		chunkSize: chunkSize,
		now:       params.Now,
	}
}

func formatExpirationTime(et picoshare.ExpirationTime) string {
	return formatTime(et.Time())
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
