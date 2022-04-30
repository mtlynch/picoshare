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
			description: "valid filename",
			input:       "dummy.png",
			output:      types.Filename("dummy.png"),
			err:         nil,
		},
		{
			description: "empty filename",
			input:       "",
			err:         parse.ErrFilenameEmpty,
		},
		{
			description: "filename with backslashes",
			input:       `filename\with\backslashes.png`,
			err:         parse.ErrFilenameIllegalCharacters,
		},
		{
			description: "filename that's just a dot",
			input:       ".",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "filename that's two dots",
			input:       "..",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "filename that's five dots",
			input:       ".....",
			err:         parse.ErrFilenameHasDotPrefix,
		},
		{
			description: "filename that's too long",
			input:       strings.Repeat("A", parse.MaxFilenameLen+1),
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
