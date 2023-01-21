package sqlite

import (
	"database/sql"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// We only store one set of settings at a time, so we used a fixed row ID.
const settingsRowID = 1

func (d db) ReadSettings() (picoshare.Settings, error) {
	var expirationInDays time.Duration
	if err := d.ctx.QueryRow(`
	SELECT
		default_expiration_in_days
	FROM
		settings
	WHERE
		id = ?`, settingsRowID).Scan(&expirationInDays); err != nil {
		if err == sql.ErrNoRows {
			return picoshare.Settings{}, nil
		}
		return picoshare.Settings{}, err
	}

	return picoshare.Settings{
		DefaultEntryLifetime: expirationInDays,
	}, nil
}

func (d db) UpdateSettings(s picoshare.Settings) error {
	_, err := d.ctx.Exec(`
	UPDATE
		settings
	SET
		default_expiration_in_days = ?
	WHERE
		id = ?`, s.DefaultEntryLifetime, settingsRowID)
	if err != nil {
		return err
	}

	return nil
}
