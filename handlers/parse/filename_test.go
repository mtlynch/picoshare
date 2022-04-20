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
		valid       bool
	}{
		{
			description: "valid filename",
			input:       "dummy.png",
			valid:       true,
			output:      types.Filename("dummy.png"),
		},
		{
			description: "filename with backslashes",
			input:       `filename\with\backslashes.png`,
			valid:       false,
		},
		{
			description: "filename that's just a dot",
			input:       ".",
			valid:       false,
		},
		{
			description: "filename that's two dots",
			input:       "..",
			valid:       false,
		},
		{
			description: "filename that's five dots",
			input:       ".....",
			valid:       false,
		},
		{
			description: "filename that's too long",
			input:       strings.Repeat("A", parse.MaxFilenameLen+1),
			valid:       false,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			filename, err := parse.Filename(tt.input)
			if tt.valid && err != nil {
				t.Fatalf("err=%v, want %v", err, !tt.valid)
			}
			if got, want := filename, tt.output; got != want {
				t.Errorf("filename=%v, want %v", filename, want)
			}
		})
	}
}
