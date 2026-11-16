package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"

	migrate "codefloe.com/mtlynch/go-evolutionary-migrate/v3"
)

//go:embed migrations/*.sql
var migrationsFs embed.FS

func applyMigrations(db *sql.DB) {
	migrationsSubFs, err := fs.Sub(migrationsFs, "migrations")
	if err != nil {
		log.Fatalf("failed to open migrations directory: %v", err)
	}

	if err := migrate.Run(context.Background(), migrate.RunParams{
		DB: db,
		FS: migrationsSubFs,
	}); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
}
