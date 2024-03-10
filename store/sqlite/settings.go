package sqlite

import (
	"database/sql"
	"log"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// We only store one set of settings at a time, so we used a fixed row ID.
const settingsRowID = 1

func (s Store) ReadSettings() (picoshare.Settings, error) {
	var expirationInDays uint16
	if err := s.ctx.QueryRow(`
   SELECT
   	default_expiration_in_days
   FROM
   	settings
   WHERE
   	id = :row_id`, sql.Named("row_id", settingsRowID)).Scan(&expirationInDays); err != nil {
		if err == sql.ErrNoRows {
			return picoshare.Settings{}, nil
		}
		return picoshare.Settings{}, err
	}

	return picoshare.Settings{
		DefaultFileLifetime: picoshare.NewFileLifetimeInDays(expirationInDays),
	}, nil
}

func (s Store) UpdateSettings(settings picoshare.Settings) error {
	log.Printf("saving new settings: %s", settings)
	expirationInDays := settings.DefaultFileLifetime.Days()
	if _, err := s.ctx.Exec(`
   UPDATE
   	settings
   SET
   	default_expiration_in_days = :expiration
   WHERE
   	id = :row_id`, sql.Named("expiration", expirationInDays), sql.Named("row_id", settingsRowID)); err != nil {
		return err
	}

	return nil
}
