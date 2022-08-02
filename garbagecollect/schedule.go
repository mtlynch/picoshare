package garbagecollect

import (
	"log"
	"sync"
	"time"

	"github.com/mtlynch/picoshare/v2/store"
)

type Scheduler struct {
	collector Collector
	ticker    *time.Ticker
	mu        sync.Mutex
}

func NewScheduler(store store.Store, interval time.Duration) Scheduler {
	return Scheduler{
		collector: NewCollector(store),
		ticker:    time.NewTicker(interval),
	}
}

func (s *Scheduler) StartAsync() {
	go func() {
		for range s.ticker.C {
			s.collect()
		}
	}()
}

func (s *Scheduler) collect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("cleaning up expired entries")
	if err := s.collector.Collect(); err != nil {
		log.Printf("garbage collection failed: %v", err)
	}
}
