package sqlite

import (
	"database/sql"
	"log"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (s Store) InsertEntryDownload(id picoshare.EntryID, r picoshare.DownloadRecord) error {
	log.Printf("recording download of file ID %s from client %s", id.String(), r.ClientIP)
	if _, err := s.ctx.Exec(`
	INSERT INTO
		downloads
	(
		entry_id,
		download_timestamp,
		client_ip,
		user_agent
	)
	VALUES(?,?,?,?)`,
		id.String(),
		formatTime(r.Time),
		r.ClientIP,
		r.UserAgent,
	); err != nil {
		log.Printf("insert into downloads table failed: %v", err)
		return err
	}
	return nil
}

func (s Store) GetEntryDownloads(id picoshare.EntryID) ([]picoshare.DownloadRecord, error) {
	rows, err := s.ctx.Query(`
	SELECT
		download_timestamp,
		client_ip,
		user_agent
	FROM
		downloads
	WHERE
		entry_id=?
	ORDER BY
		download_timestamp DESC`, id)
	if err == sql.ErrNoRows {
		return []picoshare.DownloadRecord{}, nil
	} else if err != nil {
		return []picoshare.DownloadRecord{}, err
	}

	downloads := []picoshare.DownloadRecord{}
	for rows.Next() {
		var downloadTimeRaw string
		var clientIP string
		var userAgent string

		if err := rows.Scan(&downloadTimeRaw, &clientIP, &userAgent); err != nil {
			return []picoshare.DownloadRecord{}, err
		}

		dt, err := parseDatetime(downloadTimeRaw)
		if err != nil {
			return []picoshare.DownloadRecord{}, err
		}

		downloads = append(downloads, picoshare.DownloadRecord{
			Time:      dt,
			ClientIP:  clientIP,
			UserAgent: userAgent,
		})
	}

	return downloads, nil
}
