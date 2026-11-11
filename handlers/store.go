package handlers

import (
	"context"
	"io"

	"github.com/mtlynch/picoshare/picoshare"
)

type Store interface {
	GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
	ReadEntryFile(picoshare.EntryID) (io.ReadSeeker, error)
	GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error)
	InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error
	ReserveGuestUpload(context.Context, picoshare.GuestLinkID) (picoshare.GuestUploadReservation, error)
	WriteGuestUpload(context.Context, picoshare.GuestUploadReservation, io.Reader) error
	CompleteGuestUpload(context.Context, picoshare.GuestUploadReservation, picoshare.UploadMetadata) error
	AbortGuestUpload(picoshare.GuestUploadReservation) error
	UpdateEntryMetadata(id picoshare.EntryID, metadata picoshare.UploadMetadata) error
	DeleteEntry(id picoshare.EntryID) error
	GetGuestLink(picoshare.GuestLinkID) (picoshare.GuestLink, error)
	GetGuestLinks() ([]picoshare.GuestLink, error)
	InsertGuestLink(picoshare.GuestLink) error
	DeleteGuestLink(picoshare.GuestLinkID) error
	DisableGuestLink(picoshare.GuestLinkID) error
	EnableGuestLink(picoshare.GuestLinkID) error
	InsertEntryDownload(picoshare.EntryID, picoshare.DownloadRecord) error
	GetEntryDownloads(id picoshare.EntryID) ([]picoshare.DownloadRecord, error)
	ReadSettings() (picoshare.Settings, error)
	UpdateSettings(picoshare.Settings) error
}
