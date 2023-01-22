package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func TestFriendlyName(t *testing.T) {
	for _, tt := range []struct {
		lifetime picoshare.FileLifetime
		expected string
	}{
		{
			lifetime: picoshare.NewFileLifetime(time.Hour * 24),
			expected: "1 day",
		},
		{
			lifetime: picoshare.NewFileLifetime(time.Hour * 24 * 7),
			expected: "7 days",
		},
		{
			lifetime: picoshare.NewFileLifetime(time.Hour * 24 * 30),
			expected: "30 days",
		},
		{
			lifetime: picoshare.NewFileLifetime(time.Hour * 24 * 365),
			expected: "1 year",
		},
		{
			lifetime: picoshare.NewFileLifetime(time.Hour * 24 * 365 * 10),
			expected: "10 years",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			if got, want := tt.lifetime.FriendlyName(), tt.expected; got != want {
				t.Errorf("friendlyName=%v, want=%v", got, want)
			}
		})
	}
}
