package sqlite

import (
	"context"
	"log"
	"time"
)

// Purge deletes expired entries and clears orphaned rows from the database.
func (d DB) Purge() error {
	log.Printf("deleting expired entries and orphaned data from database")
	if err := d.deleteExpiredEntries(); err != nil {
		return err
	}

	if err := d.deleteOrphanedRows(); err != nil {
		return err
	}

	return nil
}

func (d DB) Compact() error {
	log.Printf("vacuuming database")

	if _, err := d.ctx.Exec("VACUUM"); err != nil {
		log.Printf("failed to vacuum database: %v", err)
		return err
	}

	log.Printf("vacuuming complete")

	return nil
}

func (d DB) deleteExpiredEntries() error {
	log.Printf("deleting expired entries from database")

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	currentTime := formatTime(time.Now())

	_, err = tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id IN (
			SELECT
				id
			FROM
				entries
			WHERE
				entries.expiration_time IS NOT NULL AND
				entries.expiration_time < ?
		);`, currentTime)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries
	WHERE
		entries.expiration_time IS NOT NULL AND
		entries.expiration_time < ?;
	`, currentTime)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d DB) deleteOrphanedRows() error {
	log.Printf("purging orphaned rows from database")

	// Delete rows from entries_data if they don't reference valid rows in
	// entries. This can happen if the entry insertion fails partway through.
	if _, err := d.ctx.Exec(`
		DELETE FROM
			entries_data
		WHERE
		id IN (
			SELECT
				DISTINCT entries_data.id AS entry_id
			FROM
				entries_data
			LEFT JOIN
				entries ON entries_data.id = entries.id
			WHERE
				entries.id IS NULL
			)`); err != nil {
		return err
	}

	log.Printf("purge completed successfully")

	return nil
}
