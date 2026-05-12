package picoshare_test

import (
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestFileLifetime(t *testing.T) {
	for _, tt := range []struct {
		lifetime         picoshare.FileLifetime
		days             uint16
		years            uint16
		isOnYearBoundary bool
		friendlyName     string
	}{
		{
			lifetime:         picoshare.NewFileLifetimeInDays(1),
			days:             1,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "1 day",
		},
		{
			lifetime:         picoshare.NewFileLifetimeInDays(7),
			days:             7,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "7 days",
		},
		{
			lifetime:         picoshare.NewFileLifetimeInDays(30),
			days:             30,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "30 days",
		},
		{
			lifetime:         picoshare.NewFileLifetimeInYears(1),
			days:             365,
			years:            1,
			isOnYearBoundary: true,
			friendlyName:     "1 year",
		},
		{
			lifetime:         picoshare.NewFileLifetimeInDays(366),
			days:             366,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "366 days",
		},
		{
			lifetime:         picoshare.NewFileLifetimeInYears(10),
			days:             3650,
			years:            10,
			isOnYearBoundary: false,
			friendlyName:     "10 years",
		},
		{
			lifetime:         picoshare.FileLifetimeInfinite,
			isOnYearBoundary: false,
			friendlyName:     "Never",
		},
	} {
		t.Run(tt.friendlyName, func(t *testing.T) {
			if got, want := tt.lifetime.FriendlyName(), tt.friendlyName; got != want {
				t.Errorf("friendlyName=%v, want=%v", got, want)
			}
		})
	}
}
