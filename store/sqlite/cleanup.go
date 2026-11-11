package sqlite

import (
	"context"
	"database/sql"
	"log"
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

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback delete expired entries: %v", err)
		}
	}()

	currentTime := formatTime(s.now())

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

	// Incomplete guest uploads own their chunks through reservations, even
	// before they insert metadata. Only delete chunks without either owner.
	rows, err := s.db.Exec(`
		DELETE FROM entries_data
		WHERE NOT EXISTS (
			SELECT 1 FROM entries WHERE entries.id = entries_data.id
		) AND NOT EXISTS (
			SELECT 1 FROM guest_upload_reservations
			WHERE entry_id = entries_data.id
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
