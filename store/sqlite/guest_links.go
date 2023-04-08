package sqlite

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

func (d DB) GetGuestLink(id picoshare.GuestLinkID) (picoshare.GuestLink, error) {
	row := d.ctx.QueryRow(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.expiration_time AS expiration_time,
			SUM(CASE WHEN entries.id IS NOT NULL THEN 1 ELSE 0 END) AS entry_count
		FROM
			guest_links
		LEFT JOIN
			entries ON guest_links.id = entries.guest_link_id
		WHERE
			guest_links.id=?
		GROUP BY
			guest_links.id`, id)

	return guestLinkFromRow(row)
}

func (d DB) GetGuestLinks() ([]picoshare.GuestLink, error) {
	rows, err := d.ctx.Query(`
		SELECT
			guest_links.id AS id,
			guest_links.label AS label,
			guest_links.max_file_bytes AS max_file_bytes,
			guest_links.max_file_uploads AS max_file_uploads,
			guest_links.creation_time AS creation_time,
			guest_links.expiration_time AS expiration_time,
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

func (d *DB) InsertGuestLink(guestLink picoshare.GuestLink) error {
	log.Printf("saving new guest link %s", guestLink.ID)

	if _, err := d.ctx.Exec(`
	INSERT INTO guest_links
		(
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			expiration_time
		)
		VALUES (?,?,?,?,?,?)
	`, guestLink.ID, guestLink.Label, guestLink.MaxFileBytes, guestLink.MaxFileUploads, formatTime(time.Now()), formatExpirationTime(guestLink.Expires)); err != nil {
		return err
	}

	return nil
}

func (d DB) DeleteGuestLink(id picoshare.GuestLinkID) error {
	log.Printf("deleting guest link %s", id)

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		guest_links
	WHERE
		id=?`, id)
	if err != nil {
		log.Printf("deleting %s from guest_links table failed: %v", id, err)
		return err
	}

	_, err = tx.Exec(`
	UPDATE
		entries
	SET
		guest_link_id = NULL
	WHERE
		guest_link_id = ?`, id)
	if err != nil {
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
	var expirationTimeRaw string
	var filesUploaded int

	err := row.Scan(&id, &label, &maxFileBytes, &maxFileUploads, &creationTimeRaw, &expirationTimeRaw, &filesUploaded)
	if err == sql.ErrNoRows {
		return picoshare.GuestLink{}, store.GuestLinkNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.GuestLink{}, err
	}

	ct, err := parseDatetime(creationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	return picoshare.GuestLink{
		ID:             id,
		Label:          label,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
		FilesUploaded:  filesUploaded,
		Created:        ct,
		Expires:        picoshare.ExpirationTime(et),
	}, nil
}
