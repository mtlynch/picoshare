package store

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/types"
)

type Store interface {
	GetEntriesMetadata() ([]types.UploadMetadata, error)
	GetEntry(id types.EntryID) (types.UploadEntry, error)
	InsertEntry(entry types.UploadEntry) error
	DeleteEntry(id types.EntryID) error
}

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID types.EntryID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}
