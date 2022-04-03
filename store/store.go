package store

import (
	"fmt"
	"io"

	"github.com/mtlynch/picoshare/v2/types"
)

type Store interface {
	GetEntriesMetadata() ([]types.UploadMetadata, error)
	GetEntry(id types.EntryID) (types.UploadEntry, error)
	InsertEntry(reader io.Reader, metadata types.UploadMetadata) error
	DeleteEntry(id types.EntryID) error
	GetGuestLink(types.GuestLinkID) (types.GuestLink, error)
	GetGuestLinks() ([]types.GuestLink, error)
	InsertGuestLink(types.GuestLink) error
}

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID types.EntryID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}

// GuestLinkNotFoundError occurs when no guest link exists with the given ID.
type GuestLinkNotFoundError struct {
	ID types.GuestLinkID
}

func (f GuestLinkNotFoundError) Error() string {
	return fmt.Sprintf("Could not find guest link with ID %v", f.ID)
}
