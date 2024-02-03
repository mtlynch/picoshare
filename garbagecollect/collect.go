package garbagecollect

import (
	"sync"

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

	if err := c.store.Purge(); err != nil {
		return err
	}

	return nil
}
