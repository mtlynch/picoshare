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

	// Update the download count in the cache table
	if _, err := tx.Exec(`
	INSERT INTO entry_cache (entry_id, file_size, download_count, last_updated)
	VALUES (
		:entry_id,
		COALESCE((SELECT SUM(LENGTH(chunk)) FROM entries_data WHERE id = :entry_id), 0),
		1,
		:last_updated
	)
	ON CONFLICT(entry_id) DO UPDATE SET
		download_count = download_count + 1,
		last_updated = :last_updated`,
		sql.Named("entry_id", id.String()),
		sql.Named("last_updated", formatTime(r.Time)),
	); err != nil {
		log.Printf("update download count in cache failed: %v", err)
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
