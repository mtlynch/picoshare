package parse_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/internal/testutil"
	"github.com/mtlynch/picoshare/picoshare"
)

func TestExpiration(t *testing.T) {
	for _, tt := range []struct {
		description string
		currentTime time.Time
		input       string
		output      picoshare.ExpirationTime
		err         error
	}{
		{
			description: "valid expiration",
			currentTime: testutil.MustCreate2(t, time.Parse, time.DateOnly, "2024-06-01"),
			input:       "2025-01-01T00:00:00Z",
			output:      picoshare.ExpirationTime(testutil.MustCreate2(t, time.Parse, time.RFC3339, "2025-01-01T00:00:00Z")),
			err:         nil,
		},
		{
			description: "reject expiration time in the past",
			currentTime: testutil.MustCreate2(t, time.Parse, time.DateOnly, "2024-06-01"),
			input:       "2024-01-01T00:00:00Z",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationTooSoon,
		},
		{
			description: "empty string is invalid",
			currentTime: testutil.MustCreate2(t, time.Parse, time.DateOnly, "2024-06-01"),
			input:       "",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationUnrecognizedFormat,
		},
		{
			description: "string with letters causes error",
			currentTime: testutil.MustCreate2(t, time.Parse, time.DateOnly, "2024-06-01"),
			input:       "banana",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationUnrecognizedFormat,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			et, err := parse.Expiration(tt.input, tt.currentTime)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if got, want := et, tt.output; got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}
		})
	}
}
