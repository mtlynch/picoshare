package garbagecollect

import (
	"sync"
)

type (
	DatabasePurger interface {
		Purge() error
	}

	Collector struct {
		purger DatabasePurger
		mu     sync.Mutex
	}
)

func NewCollector(purger DatabasePurger) Collector {
	return Collector{
		purger: purger,
	}
}

func (c *Collector) Collect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.purger.Purge(); err != nil {
		return err
	}

	return nil
}
