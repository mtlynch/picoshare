package parse_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestFilename(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		output      types.Filename
		err         error
	}{
		{
			description: "accept valid filename",
			input:       "dummy.png",
			output:      types.Filename("dummy.png"),
			err:         nil,
		},
		{
			description: "accept filename that's the maximum length",
			input:       strings.Repeat("A", parse.MaxFilenameBytes),
			output:      types.Filename(strings.Repeat("A", parse.MaxFilenameBytes)),
			err:         nil,
		},
		{
			description: "reject empty filename",
			input:       "",
			err:         parse.ErrFilenameEmpty,
		},
		{
			description: "reject filename with backslashes",
			input:       `filename\with\backslashes.png`,
			err:         parse.ErrFilenameIllegalCharacters,
		},
		{
			description: "reject filename with forward slashes",
			input:       `filename/with/forward/slashes.png`,
			err:         parse.ErrFilenameIllegalCharacters,
		},
		{
			description: "reject filename that's just a dot",
			input:       ".",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "reject filename that's two dots",
			input:       "..",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "reject filename that's five dots",
			input:       ".....",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "reject filename that's too long",
			input:       strings.Repeat("A", parse.MaxFilenameBytes+1),
			err:         parse.ErrFilenameTooLong,
		},
		{
			description: "reject filename that's the maximum length with multibyte Unicode characters",
			input:       strings.Repeat("Ö", parse.MaxFilenameBytes),
			err:         parse.ErrFilenameTooLong,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			filename, err := parse.Filename(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", err, want)
			}
			if got, want := filename, tt.output; got != want {
				t.Errorf("filename=%v, want=%v", filename, want)
			}
		})
	}
}
