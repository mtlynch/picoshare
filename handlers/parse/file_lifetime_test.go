package parse_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
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
			output:      picoshare.NewFileLifetime(24 * time.Hour * 7),
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
