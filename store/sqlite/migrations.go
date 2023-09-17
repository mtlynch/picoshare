package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"path"
	"sort"
	"strconv"
)

type dbMigration struct {
	version int
	query   string
}

//go:embed migrations/*.sql
var migrationsFs embed.FS

func applyMigrations(ctx *sql.DB) {
	var version int
	if err := ctx.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		log.Fatalf("failed to get user_version: %v", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		log.Fatalf("error loading database migrations: %v", err)
	}

	log.Printf("migration counter: %d/%d", version, len(migrations))

	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		tx, err := ctx.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatalf("failed to create migration transaction %d: %v", migration.version, err)
		}

		if _, err := tx.Exec(migration.query); err != nil {
			log.Fatalf("failed to perform DB migration %d: %v", migration.version, err)
		}

		if _, err := tx.Exec(fmt.Sprintf(`pragma user_version=%d`, migration.version)); err != nil {
			log.Fatalf("failed to update DB version to %d: %v", migration.version, err)
		}

		if err := tx.Commit(); err != nil {
			log.Fatalf("failed to commit migration %d: %v", migration.version, err)
		}

		log.Printf("migration counter: %d/%d", migration.version, len(migrations))
	}
}

func loadMigrations() ([]dbMigration, error) {
	migrations := []dbMigration{}

	migrationsDir := "migrations"

	entries, err := migrationsFs.ReadDir(migrationsDir)
	if err != nil {
		return []dbMigration{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := migrationVersionFromFilename(entry.Name())

		query, err := migrationsFs.ReadFile(path.Join(migrationsDir, entry.Name()))
		if err != nil {
			return []dbMigration{}, err
		}

		migrations = append(migrations, dbMigration{version, string(query)})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func migrationVersionFromFilename(filename string) int {
	version, err := strconv.ParseInt(filename[:3], 10, 32)
	if err != nil {
		log.Fatalf("invalid migration number in filename: %v", filename)
	}

	return int(version)
}
