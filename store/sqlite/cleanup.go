package sqlite

import (
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

	return nil
}

func (s Store) deleteExpiredEntries() error {
	log.Printf("deleting expired entries from database")

	currentTime := formatTime(time.Now())

	if _, err := s.ctx.Exec(`
   DELETE FROM
   	entries
   WHERE
   	entries.expiration_time IS NOT NULL AND
   	entries.expiration_time < :current_time
   `, sql.Named("current_time", currentTime)); err != nil {
		return err
	}

	return nil
}
