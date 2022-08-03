package garbagecollect

import (
	"log"
	"sync"
	"time"

	"github.com/mtlynch/picoshare/v2/store"
)

type Collector struct {
	store store.Store
	mu    sync.Mutex
}

func NewCollector(store store.Store) Collector {
	return Collector{
		store: store,
	}
}

func (c *Collector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	mm, err := c.store.GetEntriesMetadata()
	if err != nil {
		return err
	}

	for _, meta := range mm {
		if time.Now().After(time.Time(meta.Expires)) {
			log.Printf("entry %v expired at %v", meta.ID, time.Time(meta.Expires).Format(time.RFC3339))
			if err := c.store.DeleteEntry(meta.ID); err != nil {
				return err
			}
		}
	}

	if err := c.store.Purge(); err != nil {
		return err
	}

	if err := c.store.Compact(); err != nil {
		return err
	}

	return nil
}
