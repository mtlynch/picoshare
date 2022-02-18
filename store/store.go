package store

import "github.com/mtlynch/picoshare/v2/types"

type Store interface {
	GetEntry(id types.EntryID) (types.UploadEntry, error)
	InsertEntry(id types.EntryID, entry types.UploadEntry) error
}
