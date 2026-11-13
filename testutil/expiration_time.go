package testutil

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
)

// MustCreateExpirationTime constructs an expiration or fails the test.
func MustCreateExpirationTime(t testing.TB, raw time.Time) picoshare.ExpirationTime {
	t.Helper()
	return MustCreate(t, picoshare.NewExpirationTime, raw)
}

// MustParseExpirationTime parses an RFC3339 expiration or fails the test.
func MustParseExpirationTime(t testing.TB, raw string) picoshare.ExpirationTime {
	t.Helper()
	parseTimestamp := func(raw string) (time.Time, error) {
		return time.Parse(time.RFC3339, raw)
	}
	timestamp := MustCreate(t, parseTimestamp, raw)
	return MustCreateExpirationTime(t, timestamp)
}
