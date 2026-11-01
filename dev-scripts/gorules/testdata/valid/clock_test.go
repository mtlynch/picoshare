package clock_test

import (
	"testing"
	"time"
)

func TestFixedTime(t *testing.T) {
	_ = time.Date(2025, time.June, 1, 12, 0, 0, 0, time.UTC)
	_ = "time.Now()"
	// Use time.Now() in production, not tests.
}
