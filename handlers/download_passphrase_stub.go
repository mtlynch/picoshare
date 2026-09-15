package handlers

import (
	"sync"

	"github.com/mtlynch/picoshare/picoshare"
)

// protectedEntries is a stand-in for persisted download passphrases. It
// remembers each protected entry's passphrase only for the lifetime of the
// process.
type protectedEntries struct {
	mu          sync.Mutex
	passphrases map[picoshare.EntryID]picoshare.DownloadPassphrase
}

func newProtectedEntries() *protectedEntries {
	return &protectedEntries{passphrases: map[picoshare.EntryID]picoshare.DownloadPassphrase{}}
}

// set records the entry's download passphrase. The empty passphrase removes
// protection from the entry.
func (p *protectedEntries) set(id picoshare.EntryID, passphrase picoshare.DownloadPassphrase) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if passphrase.Empty() {
		delete(p.passphrases, id)
	} else {
		p.passphrases[id] = passphrase
	}
}

// passphrase returns the entry's download passphrase, or the empty passphrase
// when the entry is not protected.
func (p *protectedEntries) passphrase(id picoshare.EntryID) picoshare.DownloadPassphrase {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.passphrases[id]
}

// getEntryMetadata reads the entry's metadata from the store and fills in the
// download passphrase from the in-memory map of protected entries.
func (s Server) getEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	metadata, err := s.store.GetEntryMetadata(id)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}
	metadata.DownloadPassphrase = s.protectedEntries.passphrase(id)
	return metadata, nil
}
