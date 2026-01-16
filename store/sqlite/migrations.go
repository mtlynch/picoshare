package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"

	migrate "codeberg.org/mtlynch/go-evolutionary-migrate"
)

//go:embed migrations/*.sql
var migrationsFs embed.FS

func applyMigrations(ctx *sql.DB) {
	migrationsSubFs, err := fs.Sub(migrationsFs, "migrations")
	if err != nil {
		log.Fatalf("failed to open migrations directory: %v", err)
	}

	if err := migrate.Run(context.Background(), ctx, migrationsSubFs); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
}
