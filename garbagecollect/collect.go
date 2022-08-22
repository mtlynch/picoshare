package garbagecollect

import (
	"sync"

	"github.com/mtlynch/picoshare/v2/store"
)

type Collector struct {
	store    store.Store
	vacuumDB bool
	mu       sync.Mutex
}

func NewCollector(store store.Store, vacuumDB bool) Collector {
	return Collector{
		store:    store,
		vacuumDB: vacuumDB,
	}
}

func (c *Collector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.store.Purge(); err != nil {
		return err
	}

	if c.vacuumDB {
		if err := c.store.Compact(); err != nil {
			return err
		}
	}

	return nil
}
