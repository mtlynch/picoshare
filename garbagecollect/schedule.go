package garbagecollect

import (
	"log"
	"time"

	"github.com/mtlynch/picoshare/v2/store"
)

type Scheduler struct {
	collector Collector
	interval  time.Duration
}

func NewScheduler(store store.Store, interval time.Duration) Scheduler {
	return Scheduler{
		collector: NewCollector(store),
		interval:  interval,
	}
}

func (s Scheduler) StartAsync() {
	go func() {
		for {
			log.Printf("cleaning up expired entries")
			err := s.collector.Collect()
			if err != nil {
				log.Printf("garbage collection failed: %v", err)
			}
			time.Sleep(s.interval)
		}
	}()
}
