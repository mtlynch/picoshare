package testutil

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
)

// MustParseExpirationTime parses an RFC3339 expiration or fails the test.
func MustParseExpirationTime(t testing.TB, raw string) picoshare.ExpirationTime {
	t.Helper()
	parseTimestamp := func(raw string) (time.Time, error) {
		return time.Parse(time.RFC3339, raw)
	}
	timestamp := MustCreate(t, parseTimestamp, raw)
	return picoshare.ExpirationTime(timestamp)
}
