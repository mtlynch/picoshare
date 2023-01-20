package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (d db) ReadSettings() (picoshare.Settings, error) {
	var expirationInDays time.Duration
	if err := d.ctx.QueryRow(`
	SELECT
		default_expiration_in_days
	FROM
		settings`).Scan(&expirationInDays); err != nil {
		if err == sql.ErrNoRows {
			return picoshare.Settings{}, nil
		}
		return picoshare.Settings{}, err
	}

	return picoshare.Settings{
		DefaultEntryLifetime: expirationInDays,
	}, nil
}

func (d db) UpdateSettings(picoshare.Settings) error {
	return errors.New("not implemented")
}
