package parse_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
)

func TestExpiration(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		output      picoshare.ExpirationTime
		err         error
	}{
		{
			description: "valid expiration",
			input:       "2025-01-01T00:00:00Z",
			output:      mustParseExpiration("2025-01-01T00:00:00Z"),
			err:         nil,
		},
		{
			description: "reject expiration time in the past",
			input:       "2000-01-01T00:00:00Z",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationTooSoon,
		},
		{
			description: "empty string is invalid",
			input:       "",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationUnrecognizedFormat,
		},
		{
			description: "string with letters causes error",
			input:       "banana",
			output:      picoshare.ExpirationTime{},
			err:         parse.ErrExpirationUnrecognizedFormat,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			et, err := parse.Expiration(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if got, want := et, tt.output; got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}
		})
	}
}

func mustParseExpiration(s string) picoshare.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return picoshare.ExpirationTime(et)
}
