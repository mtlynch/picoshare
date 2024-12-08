package sqlite

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

func (s Store) GetGuestLink(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
	row := s.ctx.QueryRow(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.url_expiration_time AS url_expiration_time,
			guest_links.file_expiration_time AS file_expiration_time,
			SUM(CASE WHEN entries.id IS NOT NULL THEN 1 ELSE 0 END) AS entry_count
		FROM
			guest_links
		LEFT JOIN
			entries ON guest_links.id = entries.guest_link_id
		WHERE
			guest_links.id=:id
		GROUP BY
			guest_links.id`, sql.Named("id", id))

	return guestLinkFromRow(row)
}

func (s Store) GetGuestLinks() ([]picoshare.GuestLink, error) {
	rows, err := s.ctx.Query(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.url_expiration_time AS url_expiration_time,
			guest_links.file_expiration_time AS file_expiration_time,
			SUM(CASE WHEN entries.id IS NOT NULL THEN 1 ELSE 0 END) AS entry_count
		FROM
			guest_links
		LEFT JOIN
			entries ON guest_links.id = entries.guest_link_id
		GROUP BY
			guest_links.id`)
	if err != nil {
		return []picoshare.GuestLink{}, err
	}

	gls := []picoshare.GuestLink{}
	for rows.Next() {
		gl, err := guestLinkFromRow(rows)
		if err != nil {
			return []picoshare.GuestLink{}, err
		}

		gls = append(gls, gl)
	}

	return gls, nil
}

func (s *Store) InsertGuestLink(guestLink picoshare.GuestLink) error {
	log.Printf("saving new guest link %s", guestLink.ID)

	if _, err := s.ctx.Exec(`
	INSERT INTO guest_links
		(
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			url_expiration_time,
			file_expiration_time
		)
		VALUES (:id, :label, :max_file_bytes, :max_file_uploads, :creation_time, :url_expiration_time, :file_expiration_time)
	`,
		sql.Named("id", guestLink.ID),
		sql.Named("label", guestLink.Label),
		sql.Named("max_file_bytes", guestLink.MaxFileBytes),
		sql.Named("max_file_uploads", guestLink.MaxFileUploads),
		sql.Named("creation_time", formatTime(time.Now())),
		sql.Named("url_expiration_time", formatExpirationTime(guestLink.UrlExpires)),
		sql.Named("file_expiration_time", guestLink.FileLifetime.Duration().Hours())); err != nil {
		return err
	}

	return nil
}

func (s Store) DeleteGuestLink(id picoshare.GuestLinkID) error {
	log.Printf("deleting guest link %s", id)

	tx, err := s.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	if _, err = tx.Exec(`
	DELETE FROM
		guest_links
	WHERE
		id=:id`, sql.Named("id", id)); err != nil {
		log.Printf("deleting %s from guest_links table failed: %v", id, err)
		return err
	}

	if _, err = tx.Exec(`
	UPDATE
		entries
	SET
		guest_link_id = NULL
	WHERE
		guest_link_id = :id`, sql.Named("id", id)); err != nil {
		log.Printf("removing references to guest link %s from entries table failed: %v", id, err)
		return err
	}

	return tx.Commit()
}

func guestLinkFromRow(row rowScanner) (picoshare.GuestLink, error) {
	var id picoshare.GuestLinkID
	var label picoshare.GuestLinkLabel
	var maxFileBytes picoshare.GuestUploadMaxFileBytes
	var maxFileUploads picoshare.GuestUploadCountLimit
	var creationTimeRaw string
	var urlExpirationTimeRaw string
	var fileLifetimeRaw *string
	var filesUploaded int

	err := row.Scan(&id, &label, &maxFileBytes, &maxFileUploads, &creationTimeRaw, &urlExpirationTimeRaw, &fileLifetimeRaw, &filesUploaded)
	if err == sql.ErrNoRows {
		return picoshare.GuestLink{}, store.GuestLinkNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.GuestLink{}, err
	}

	ct, err := parseDatetime(creationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	uet, err := parseDatetime(urlExpirationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	var fileLifetime picoshare.FileLifetime
	if fileLifetimeRaw == nil {
		fileLifetime = picoshare.FileLifetimeInfinite
	} else {
		dur, err := parseFileDatetime(*fileLifetimeRaw)
		if err != nil {
			return picoshare.GuestLink{}, err
		}
		fileLifetime = picoshare.NewFileLifetimeFromDuration(dur)
	}

	return picoshare.GuestLink{
		ID:             id,
		Label:          label,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
		FilesUploaded:  filesUploaded,
		Created:        ct,
		UrlExpires:     picoshare.ExpirationTime(uet),
		FileLifetime:   fileLifetime,
	}, nil
}
