package sqlite

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// Purge deletes expired entries and clears orphaned rows from the database.
func (s Store) Purge() error {
	log.Printf("deleting expired entries and orphaned data from database")
	if err := s.deleteExpiredEntries(); err != nil {
		return err
	}

	if err := s.deleteOrphanedRows(); err != nil {
		return err
	}

	return nil
}

func (s Store) deleteExpiredEntries() error {
	log.Printf("deleting expired entries from database")

	tx, err := s.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback delete expired entries: %v", err)
		}
	}()

	currentTime := formatTime(time.Now())

	// Delete download records for expired entries first to avoid foreign key
	// constraint violations.
	if _, err = tx.Exec(`
   DELETE FROM
   	downloads
   WHERE
   	entry_id IN (
   		SELECT
   			id
   		FROM
   			entries
   		WHERE
   			entries.expiration_time IS NOT NULL AND
   			entries.expiration_time < :current_time
   	);`, sql.Named("current_time", currentTime)); err != nil {
		return err
	}

	if _, err = tx.Exec(`
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
   			entries.expiration_time < :current_time
   	);`, sql.Named("current_time", currentTime)); err != nil {
		return err
	}

	if _, err = tx.Exec(`
   DELETE FROM
   	entries
   WHERE
   	entries.expiration_time IS NOT NULL AND
   	entries.expiration_time < :current_time;
   `, sql.Named("current_time", currentTime)); err != nil {
		return err
	}

	return tx.Commit()
}

func (s Store) deleteOrphanedRows() error {
	log.Printf("purging orphaned rows from database")

	// Delete rows from entries_data if they don't reference valid rows in
	// entries. This can happen if the entry insertion fails partway through.
	rows, err := s.ctx.Exec(`
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
   		)`)
	if err != nil {
		return err
	}

	ra, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	log.Printf("purge completed successfully (%d rows affected)", ra)

	return nil
}
