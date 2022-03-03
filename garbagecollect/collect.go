package garbagecollect

import (
	"time"

	"github.com/mtlynch/picoshare/v2/store"
)

type Collector struct {
	store store.Store
}

func New(store store.Store) Collector {
	return Collector{
		store: store,
	}
}

func (c Collector) Collect() error {
	mm, err := c.store.GetEntriesMetadata()
	if err != nil {
		return err
	}

	for _, meta := range mm {
		if time.Now().After(time.Time(meta.Expires)) {
			if err := c.store.DeleteEntry(meta.ID); err != nil {
				return err
			}
		}
	}

	return nil
}
