// The memory package is an in-memory implementation of the store.Store
// interface. It aids in testing but is not ideal for production usage.

package memory

import (
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

type memstore struct {
	entries map[types.EntryID]types.UploadEntry
}

func New() store.Store {
	return &memstore{
		entries: map[types.EntryID]types.UploadEntry{},
	}
}

func (m memstore) GetEntriesMetadata() ([]types.UploadMetadata, error) {
	ee := []types.UploadMetadata{}
	for _, e := range m.entries {
		ee = append(ee, e.UploadMetadata)
	}
	return ee, nil
}

func (m memstore) GetEntry(id types.EntryID) (types.UploadEntry, error) {
	if entry, ok := m.entries[id]; ok {
		return entry, nil
	} else {
		return types.UploadEntry{}, store.EntryNotFoundError{ID: id}
	}
}

func (m *memstore) InsertEntry(entry types.UploadEntry) error {
	m.entries[entry.ID] = entry
	return nil
}
