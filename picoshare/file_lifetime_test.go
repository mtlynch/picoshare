package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/testutil"
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

func TestFileLifetimeExpirationFromTime(t *testing.T) {
	for _, tt := range []struct {
		explanation        string
		lifetime           picoshare.FileLifetime
		now                time.Time
		expirationExpected picoshare.ExpirationTime
		errExpected        bool
	}{
		{
			explanation:        "a finite lifetime produces an expiration after the starting time",
			lifetime:           picoshare.NewFileLifetimeInDays(1),
			now:                time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			expirationExpected: testutil.MustCreate(t, picoshare.NewExpirationTime, time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)),
		},
		{
			explanation:        "an expiration at the PicoShare epoch is accepted",
			lifetime:           picoshare.NewFileLifetimeInDays(1),
			now:                time.Date(2022, time.February, 19, 0, 0, 0, 0, time.UTC),
			expirationExpected: testutil.MustCreate(t, picoshare.NewExpirationTime, time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC)),
		},
		{
			explanation: "an expiration before the PicoShare epoch is rejected",
			lifetime:    picoshare.NewFileLifetimeInDays(1),
			now:         time.Date(2022, time.February, 18, 0, 0, 0, 0, time.UTC),
			errExpected: true,
		},
		{
			explanation:        "an infinite lifetime preserves the never-expire sentinel",
			lifetime:           picoshare.FileLifetimeInfinite,
			now:                time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			expirationExpected: picoshare.NeverExpire,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			expiration, err := tt.lifetime.ExpirationFromTime(tt.now)
			if got, want := err != nil, tt.errExpected; got != want {
				t.Fatalf("error=%v, want error=%v", err, want)
			}
			if got, want := expiration, tt.expirationExpected; got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}
		})
	}
}
