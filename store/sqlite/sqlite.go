package sqlite

import (
	"database/sql"
	"log"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/blobio"
	"github.com/tetratelabs/wazero"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

const (
	timeFormat = time.RFC3339
)

type (
	Store struct {
		ctx *sql.DB
	}

	rowScanner interface {
		Scan(...interface{}) error
	}
)

func New(path string, optimizeForLitestream bool) Store {
	log.Printf("reading DB from %s", path)
	ctx, err := driver.Open(path, blobio.Register)
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

	// Try to prevent memory exhaustion.
	// https://github.com/ncruces/go-sqlite3/issues/148#issuecomment-2344857749
	sqlite3.RuntimeConfig = wazero.NewRuntimeConfig().WithMemoryLimitPages(512)
	ctx.SetMaxOpenConns(1)

	applyMigrations(ctx)

	return Store{
		ctx: ctx,
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
