package picoshare_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestEntryIDFromString(t *testing.T) {
	for _, tt := range []struct {
		explanation     string
		input           string
		isValidExpected bool
	}{
		{
			explanation:     "ten allowed characters are valid",
			input:           strings.Repeat("a", picoshare.EntryIDLength),
			isValidExpected: true,
		},
		{
			explanation:     "an ID shorter than ten characters is invalid",
			input:           strings.Repeat("a", picoshare.EntryIDLength-1),
			isValidExpected: false,
		},
		{
			explanation:     "an ID longer than ten characters is invalid",
			input:           strings.Repeat("a", picoshare.EntryIDLength+1),
			isValidExpected: false,
		},
		{
			explanation:     "a visually ambiguous uppercase I is invalid",
			input:           strings.Repeat("a", picoshare.EntryIDLength-1) + "I",
			isValidExpected: false,
		},
		{
			explanation:     "a visually ambiguous lowercase l is invalid",
			input:           strings.Repeat("a", picoshare.EntryIDLength-1) + "l",
			isValidExpected: false,
		},
		{
			explanation:     "a Unicode character is invalid",
			input:           strings.Repeat("a", picoshare.EntryIDLength-1) + "é",
			isValidExpected: false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			id, err := picoshare.EntryIDFromString(tt.input)
			isValid := err == nil
			if got, want := isValid, tt.isValidExpected; got != want {
				t.Fatalf("EntryIDFromString validity=%t, want=%t", got, want)
			}
			if !isValid {
				return
			}
			if got, want := id.String(), tt.input; got != want {
				t.Errorf("ID=%q, want=%q", got, want)
			}
		})
	}
}

func TestNewEntryID(t *testing.T) {
	id := picoshare.NewEntryID()
	if got, want := len(id.String()), picoshare.EntryIDLength; got != want {
		t.Errorf("length=%d, want=%d", got, want)
	}

	_, err := picoshare.EntryIDFromString(id.String())
	if err != nil {
		t.Errorf("failed to parse generated entry ID: %v", err)
	}
}
