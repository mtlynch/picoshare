package parse_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestFilename(t *testing.T) {
	var tests = []struct {
		description    string
		filename       string
		validExpected  bool
		parsedExpected types.Filename
	}{
		{
			description:    "valid filename",
			filename:       "dummy.png",
			validExpected:  true,
			parsedExpected: types.Filename("dummy.png"),
		},
		{
			description:   "filename with backslashes",
			filename:      `filename\with\backslashes.png`,
			validExpected: false,
		},
		{
			description:   "filename that's just a dot",
			filename:      ".",
			validExpected: false,
		},
		{
			description:   "filename that's two dots",
			filename:      "..",
			validExpected: false,
		},
		{
			description:   "filename that's five dots",
			filename:      ".....",
			validExpected: false,
		},
		{
			description:   "filename that's too long",
			filename:      strings.Repeat("A", parse.MaxFilenameLen+1),
			validExpected: false,
		},
	}

	for _, tt := range tests {
		parsedActual, errActual := parse.Filename(tt.filename)
		if (errActual == nil) != tt.validExpected {
			t.Errorf("%s: input [%s], got %v, want %v", tt.description, tt.filename, errActual, tt.validExpected)
		} else if parsedActual != tt.parsedExpected {
			t.Errorf("%s: input [%s], got %v, want %v", tt.description, tt.filename, parsedActual, tt.parsedExpected)
		}
	}
}
