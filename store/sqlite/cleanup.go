package sqlite

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// Purge deletes expired entries and clears orphaned rows from the database.
func (s Store) Purge() error {
	log.Printf("deleting expired entries from database")
	if err := s.deleteExpiredEntries(); err != nil {
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
		if err := tx.Rollback(); err != nil {
			log.Printf("failed to rollback delete expired entries: %v", err)
		}
	}()

	currentTime := formatTime(time.Now())

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
