package parse_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
)

func TestFileNote(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		valid       bool
		output      picoshare.FileNote
	}{
		{
			description: "valid message",
			input:       "Shared with my college group chat",
			valid:       true,
			output:      makeFileNote("Shared with my college group chat"),
		},
		{
			description: "message of maximum bytes",
			input:       strings.Repeat("A", parse.MaxFileNoteBytes),
			valid:       true,
			output:      makeFileNote(strings.Repeat("A", parse.MaxFileNoteBytes)),
		},
		{
			description: "message of maximum length with multibyte Unicode characters",
			input:       strings.Repeat("Ö", parse.MaxFileNoteBytes),
			valid:       false,
			output:      picoshare.FileNote{},
		},
		{
			description: "empty note",
			input:       "",
			valid:       true,
			output:      picoshare.FileNote{},
		},
		{
			description: "note that's too long",
			input:       strings.Repeat("A", parse.MaxFileNoteBytes+1),
			valid:       false,
		},
		{
			description: "literal null string",
			input:       "null",
			valid:       false,
		},
		{
			description: "literal undefined string",
			input:       "undefined",
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
			if got, want := note.String(), tt.output.String(); got != want {
				t.Errorf("note=%v, want=%v", note, want)
			}
		})
	}
}

func makeFileNote(s string) picoshare.FileNote {
	return picoshare.FileNote{Value: &s}
}
