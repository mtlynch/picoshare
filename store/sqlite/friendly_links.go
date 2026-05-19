package sqlite

import (
	"database/sql"
	"log"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

func (s Store) GetFriendlyLink(name picoshare.FriendlyName) (picoshare.FriendlyLink, error) {
	row := s.ctx.QueryRow(`
		SELECT
			friendly_name,
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			url_expiration_time,
			file_expiration_time,
			is_disabled
		FROM
			friendly_links
		WHERE
			friendly_name=:name`, sql.Named("name", name))

	return friendlyLinkFromRow(row, name)
}

func (s Store) GetFriendlyLinks() ([]picoshare.FriendlyLink, error) {
	rows, err := s.ctx.Query(`
		SELECT
			friendly_name,
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			url_expiration_time,
			file_expiration_time,
			is_disabled
		FROM
			friendly_links`)
	if err != nil {
		return []picoshare.FriendlyLink{}, err
	}

	fls := []picoshare.FriendlyLink{}
	for rows.Next() {
		fl, err := friendlyLinkFromRow(rows, "")
		if err != nil {
			return []picoshare.FriendlyLink{}, err
		}

		fls = append(fls, fl)
	}

	return fls, nil
}

func (s *Store) InsertFriendlyLink(fl picoshare.FriendlyLink) error {
	log.Printf("saving new friendly link %s -> %s", fl.FriendlyName, fl.EntryID)

	if _, err := s.ctx.Exec(`
	INSERT INTO friendly_links
		(
			friendly_name,
			id,
			label,
			max_file_bytes,
			max_file_uploads,
			creation_time,
			url_expiration_time,
			file_expiration_time,
			is_disabled
		)
		VALUES (:friendly_name, :id, :label, :max_file_bytes, :max_file_uploads, :creation_time, :url_expiration_time, :file_expiration_time, :is_disabled)
	`,
		sql.Named("friendly_name", fl.FriendlyName),
		sql.Named("id", fl.EntryID),
		sql.Named("label", fl.Label),
		sql.Named("max_file_bytes", fl.MaxFileBytes),
		sql.Named("max_file_uploads", fl.MaxFileUploads),
		sql.Named("creation_time", formatTime(fl.Created)),
		sql.Named("url_expiration_time", formatExpirationTime(fl.UrlExpires)),
		sql.Named("file_expiration_time", formatFileLifetime(fl.MaxFileLifetime)),
		sql.Named("is_disabled", fl.IsDisabled)); err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateFriendlyLink(fl picoshare.FriendlyLink) error {
	log.Printf("updating friendly link %s -> %s", fl.FriendlyName, fl.EntryID)

	if _, err := s.ctx.Exec(`
	UPDATE friendly_links
	SET
		id = :id,
		label = :label,
		max_file_bytes = :max_file_bytes,
		max_file_uploads = :max_file_uploads,
		url_expiration_time = :url_expiration_time,
		file_expiration_time = :file_expiration_time,
		is_disabled = :is_disabled
	WHERE
		friendly_name = :friendly_name
	`,
		sql.Named("friendly_name", fl.FriendlyName),
		sql.Named("id", fl.EntryID),
		sql.Named("label", fl.Label),
		sql.Named("max_file_bytes", fl.MaxFileBytes),
		sql.Named("max_file_uploads", fl.MaxFileUploads),
		sql.Named("url_expiration_time", formatExpirationTime(fl.UrlExpires)),
		sql.Named("file_expiration_time", formatFileLifetime(fl.MaxFileLifetime)),
		sql.Named("is_disabled", fl.IsDisabled)); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteFriendlyLink(name picoshare.FriendlyName) error {
	log.Printf("deleting friendly link %s", name)

	if _, err := s.ctx.Exec(`
	DELETE FROM
		friendly_links
	WHERE
		friendly_name=:name`, sql.Named("name", name)); err != nil {
		log.Printf("deleting %s from friendly_links table failed: %v", name, err)
		return err
	}

	return nil
}

func friendlyLinkFromRow(row rowScanner, name picoshare.FriendlyName) (picoshare.FriendlyLink, error) {
	var friendlyName picoshare.FriendlyName
	var id picoshare.EntryID
	var label picoshare.GuestLinkLabel
	var maxFileBytes picoshare.GuestUploadMaxFileBytes
	var maxFileUploads picoshare.GuestUploadCountLimit
	var creationTimeRaw string
	var urlExpirationTimeRaw string
	var fileLifetimeRaw *string
	var isDisabled bool

	err := row.Scan(&friendlyName, &id, &label, &maxFileBytes, &maxFileUploads, &creationTimeRaw, &urlExpirationTimeRaw, &fileLifetimeRaw, &isDisabled)
	if err == sql.ErrNoRows {
		return picoshare.FriendlyLink{}, store.FriendlyLinkNotFoundError{Name: string(name)}
	} else if err != nil {
		return picoshare.FriendlyLink{}, err
	}

	ct, err := parseDatetime(creationTimeRaw)
	if err != nil {
		return picoshare.FriendlyLink{}, err
	}

	uet, err := parseDatetime(urlExpirationTimeRaw)
	if err != nil {
		return picoshare.FriendlyLink{}, err
	}

	var fileLifetime picoshare.FileLifetime
	if fileLifetimeRaw == nil {
		fileLifetime = picoshare.FileLifetimeInfinite
	} else {
		fileLifetime, err = parseFileLifetime(*fileLifetimeRaw)
		if err != nil {
			return picoshare.FriendlyLink{}, err
		}
	}

	return picoshare.FriendlyLink{
		FriendlyName:    friendlyName,
		EntryID:         id,
		Label:           label,
		Created:         ct,
		UrlExpires:      picoshare.ExpirationTime(uet),
		MaxFileLifetime: fileLifetime,
		MaxFileBytes:    maxFileBytes,
		MaxFileUploads:  maxFileUploads,
		IsDisabled:      isDisabled,
	}, nil
}
