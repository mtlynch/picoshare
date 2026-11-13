package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/testutil"
)

func TestNewExpirationTime(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		input       time.Time
		errExpected bool
	}{
		{
			explanation: "zero time is before the PicoShare epoch and is rejected",
			input:       time.Time{},
			errExpected: true,
		},
		{
			explanation: "one nanosecond before the PicoShare epoch is rejected",
			input:       time.Date(2022, time.February, 19, 23, 59, 59, 999999999, time.UTC),
			errExpected: true,
		},
		{
			explanation: "the PicoShare epoch itself is accepted",
			input:       time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			explanation: "one nanosecond after the PicoShare epoch is accepted",
			input:       time.Date(2022, time.February, 20, 0, 0, 0, 1, time.UTC),
		},
		{
			explanation: "historical expirations after the PicoShare epoch are accepted",
			input:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			explanation: "the never-expire timestamp is accepted",
			input:       picoshare.NeverExpire.Time(),
		},
		{
			explanation: "the epoch in another time zone is accepted",
			input:       time.Date(2022, time.February, 19, 19, 0, 0, 0, time.FixedZone("UTC-5", -5*60*60)),
		},
		{
			explanation: "a local date after the epoch with an earlier UTC instant is rejected",
			input:       time.Date(2022, time.February, 20, 0, 0, 0, 0, time.FixedZone("UTC+1", 60*60)),
			errExpected: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			expiration, err := picoshare.NewExpirationTime(tt.input)
			if got, want := err != nil, tt.errExpected; got != want {
				t.Fatalf("error=%v, want error=%v", err, want)
			}
			if tt.errExpected {
				if got, want := expiration, (picoshare.ExpirationTime{}); got != want {
					t.Errorf("expiration=%v, want=%v", got, want)
				}
				return
			}
			if got, want := expiration.Time(), tt.input; got != want {
				t.Errorf("expiration.Time()=%v, want=%v", got, want)
			}
		})
	}
}

func TestExpirationTimeString(t *testing.T) {
	timestamp := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	expiration := testutil.MustCreate(t, picoshare.NewExpirationTime, timestamp)
	if got, want := expiration.String(), timestamp.String(); got != want {
		t.Errorf("expiration.String()=%q, want=%q", got, want)
	}
}

func TestExpirationTimeAfter(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		expiration    picoshare.ExpirationTime
		other         picoshare.ExpirationTime
		afterExpected bool
	}{
		{
			explanation:   "an expiration one nanosecond later is after the other expiration",
			expiration:    testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 1, time.UTC)),
			other:         testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
			afterExpected: true,
		},
		{
			explanation: "an earlier expiration is not after the other expiration",
			expiration:  testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
			other:       testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 1, time.UTC)),
		},
		{
			explanation: "equal expirations are not after each other",
			expiration:  testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
			other:       testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			explanation: "equal instants in different time zones are not after each other",
			expiration:  testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 1, 0, 0, 0, time.FixedZone("UTC+1", 60*60))),
			other:       testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			explanation:   "the never-expire timestamp sorts after an earlier expiration",
			expiration:    picoshare.NeverExpire,
			other:         testutil.MustCreateExpirationTime(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)),
			afterExpected: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := tt.expiration.After(tt.other), tt.afterExpected; got != want {
				t.Errorf("expiration.After(other)=%v, want=%v", got, want)
			}
		})
	}
}
