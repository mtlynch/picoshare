package parse_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
)

func TestGuestLinkLabel(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		output      picoshare.GuestLinkLabel
		err         error
	}{
		{
			description: "accept valid label",
			input:       "For my good pals",
			output:      picoshare.GuestLinkLabel("For my good pals"),
			err:         nil,
		},
		{
			description: "allow empty label",
			input:       "",
			output:      picoshare.GuestLinkLabel(""),
			err:         nil,
		},
		{
			description: "reject labels that are too long",
			input:       strings.Repeat("A", parse.MaxGuestLinkLabelLength+1),
			output:      picoshare.GuestLinkLabel(""),
			err:         parse.ErrGuestLinkLabelTooLong,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			label, err := parse.GuestLinkLabel(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", err, want)
			}
			if got, want := label, tt.output; got != want {
				t.Errorf("label=%v, want=%v", label, want)
			}
		})
	}
}
