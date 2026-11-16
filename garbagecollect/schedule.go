package garbagecollect

import (
	"log"
	"time"
)

type Scheduler struct {
	collector *Collector
	ticker    *time.Ticker
}

func NewScheduler(collector *Collector, interval time.Duration) Scheduler {
	return Scheduler{
		collector: collector,
		ticker:    time.NewTicker(interval),
	}
}

func (s *Scheduler) StartAsync() {
	go func() {
		for {
			log.Printf("performing database maintenance")
			if err := s.collector.Collect(); err != nil {
				log.Printf("database maintenance failed: %v", err)
			}
			<-s.ticker.C
		}
	}()
}
