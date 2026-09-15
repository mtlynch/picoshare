package picoshare_test

import (
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestNewEntryID(t *testing.T) {
	for _, tt := range []struct {
		explanation     string
		input           string
		isValidExpected bool
	}{
		{
			explanation:     "ten allowed characters are valid",
			input:           "aA23456789",
			isValidExpected: true,
		},
		{
			explanation:     "an ID shorter than ten characters is invalid",
			input:           "aA2345678",
			isValidExpected: false,
		},
		{
			explanation:     "an ID longer than ten characters is invalid",
			input:           "aA23456789a",
			isValidExpected: false,
		},
		{
			explanation:     "a visually ambiguous uppercase I is invalid",
			input:           "aA2345678I",
			isValidExpected: false,
		},
		{
			explanation:     "a visually ambiguous lowercase l is invalid",
			input:           "aA2345678l",
			isValidExpected: false,
		},
		{
			explanation:     "a Unicode character is invalid",
			input:           "aA2345678é",
			isValidExpected: false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			id, err := picoshare.NewEntryID(tt.input)
			isValid := err == nil
			if got, want := isValid, tt.isValidExpected; got != want {
				t.Fatalf("NewEntryID validity=%t, want=%t", got, want)
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
