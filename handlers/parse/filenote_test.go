package parse_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestFileNote(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		valid       bool
		output      types.FileNote
	}{
		{
			description: "valid message",
			input:       "Shared with my college group chat",
			valid:       true,
			output:      makeFileNote("Shared with my college group chat"),
		},
		{
			description: "note that's too long",
			input:       strings.Repeat("A", parse.MaxFileNoteLen+1),
			valid:       false,
		},
		{
			description: "contains a <script> tag",
			input:       "<script>alert(1)</script>",
			valid:       false,
		},
		{
			description: "contains a <script> tag with extra whitespace",
			input:       "< \n\t script  >alert(1)</ \n\tscript >",
			valid:       false,
		},
		{
			description: "contains an <iframe> tag with extra whitespace",
			input:       "< \n\t iframe  >foo</ \n\tiframe >",
			valid:       false,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			note, err := parse.FileNote(tt.input)
			if tt.valid != (err == nil) {
				t.Fatalf("err=%v, want=%v", err, !tt.valid)
			}
			if !tt.valid {
				return
			}
			if got, want := note, tt.output; got == nil || *got != *want {
				t.Errorf("note=%v, want=%v", *note, *want)
			}
		})
	}
}

func makeFileNote(s string) types.FileNote {
	return types.FileNote(&s)
}
