package garbagecollect

import (
	"log"
	"time"

	"github.com/mtlynch/picoshare/v2/store"
)

type Scheduler struct {
	collector Collector
	ticker    *time.Ticker
}

func NewScheduler(store store.Store, interval time.Duration) Scheduler {
	return Scheduler{
		collector: NewCollector(store),
		ticker:    time.NewTicker(interval),
	}
}

func (s Scheduler) StartAsync() {
	go func() {
		for range s.ticker.C {
			log.Printf("cleaning up expired entries")
			err := s.collector.Collect()
			if err != nil {
				log.Printf("garbage collection failed: %v", err)
			}
		}
	}()
}
