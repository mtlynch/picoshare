package sqlite

import (
	"log"
	"path"
	"sort"
	"strconv"
)

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
