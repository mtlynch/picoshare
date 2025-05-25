package sqlite

import (
	"context"
	"database/sql"
	"log"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (s Store) InsertEntryDownload(id picoshare.EntryID, r picoshare.DownloadRecord) error {
	log.Printf("recording download of file ID %s from client %s", id.String(), r.ClientIP)

	// Use a transaction to ensure both operations succeed or fail together
	tx, err := s.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback download insert: %v", err)
		}
	}()

	// Insert the download record
	if _, err := tx.Exec(`
	INSERT INTO
		downloads
	(
		entry_id,
		download_timestamp,
		client_ip,
		user_agent
	)
	VALUES(:entry_id, :download_timestamp, :client_ip, :user_agent)`,
		sql.Named("entry_id", id.String()),
		sql.Named("download_timestamp", formatTime(r.Time)),
		sql.Named("client_ip", r.ClientIP),
		sql.Named("user_agent", r.UserAgent),
	); err != nil {
		log.Printf("insert into downloads table failed: %v", err)
		return err
	}

	// Increment the download count in the entries table
	if _, err := tx.Exec(`
	UPDATE entries
	SET download_count = COALESCE(download_count, 0) + 1
	WHERE id = :entry_id`,
		sql.Named("entry_id", id.String()),
	); err != nil {
		log.Printf("update download count failed: %v", err)
		return err
	}

	return tx.Commit()
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
		entry_id=:entry_id
	ORDER BY
		download_timestamp DESC`, sql.Named("entry_id", id))
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
