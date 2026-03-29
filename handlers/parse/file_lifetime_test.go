package parse_test

import (
	"testing"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
)

func TestFileLifetime(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       uint16
		output      picoshare.FileLifetime
		err         error
	}{
		{
			description: "valid lifetime",
			input:       7,
			output:      picoshare.NewFileLifetimeInDays(7),
			err:         nil,
		},
		{
			description: "accepts the minimum valid lifetime",
			input:       1,
			output:      picoshare.NewFileLifetimeInDays(1),
			err:         nil,
		},
		{
			description: "accepts the maximum valid lifetime",
			input:       365 * 10,
			output:      picoshare.NewFileLifetimeInYears(10),
			err:         nil,
		},
		{
			description: "rejects too short a lifetime",
			input:       0,
			output:      picoshare.FileLifetime{},
			err:         parse.ErrFileLifetimeTooShort,
		},
		{
			description: "rejects too long a lifetime",
			input:       3651,
			output:      picoshare.FileLifetime{},
			err:         parse.ErrFileLifetimeTooLong,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			lt, err := parse.FileLifetime(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if got, want := lt, tt.output; got != want {
				t.Errorf("lifetime=%s, want=%s", got.FriendlyName(), want.FriendlyName())
			}
		})
	}
}
