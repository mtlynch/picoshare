package garbagecollect_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/garbagecollect"
)

func TestSchedulerCollectsBeforeFirstTick(t *testing.T) {
	purger := notifyingPurger{purged: make(chan struct{}, 1)}
	collector := garbagecollect.NewCollector(purger)
	interval := 7 * time.Hour
	scheduler := garbagecollect.NewScheduler(&collector, interval)
	scheduler.StartAsync()

	select {
	case <-purger.purged:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not purge before the first scheduled tick")
	}
}

type notifyingPurger struct {
	purged chan struct{}
}

func (p notifyingPurger) Purge() error {
	p.purged <- struct{}{}
	return nil
}
