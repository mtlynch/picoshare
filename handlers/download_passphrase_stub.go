package handlers

import (
	"sync"

	"github.com/mtlynch/picoshare/picoshare"
)

// protectedEntries is a stand-in for persisted download passphrases. It
// remembers which entries require a passphrase only for the lifetime of the
// process, and every protected entry uses the same fixed passphrase.
type protectedEntries struct {
	mu  sync.Mutex
	ids map[picoshare.EntryID]struct{}
}

func newProtectedEntries() *protectedEntries {
	return &protectedEntries{ids: map[picoshare.EntryID]struct{}{}}
}

// set records whether the entry requires a passphrase to download.
func (p *protectedEntries) set(id picoshare.EntryID, protected bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if protected {
		p.ids[id] = struct{}{}
	} else {
		delete(p.ids, id)
	}
}

// passphrase returns the fixed passphrase when the entry is protected, or the
// empty passphrase otherwise.
func (p *protectedEntries) passphrase(id picoshare.EntryID) picoshare.DownloadPassphrase {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.ids[id]; !ok {
		return picoshare.DownloadPassphrase{}
	}
	return stubDownloadPassphrase()
}

// stubDownloadPassphrase returns the passphrase that every protected entry
// uses until the store persists per-entry passphrases.
func stubDownloadPassphrase() picoshare.DownloadPassphrase {
	passphrase, err := picoshare.NewDownloadPassphrase("test")
	if err != nil {
		panic(err)
	}
	return passphrase
}

// getEntryMetadata reads the entry's metadata from the store and fills in the
// download passphrase from the in-memory set of protected entries.
func (s Server) getEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error) {
	metadata, err := s.store.GetEntryMetadata(id)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}
	metadata.DownloadPassphrase = s.protectedEntries.passphrase(id)
	return metadata, nil
}
