package picoshare_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestNewPassphrase(t *testing.T) {
	for _, tt := range []struct {
		explanation     string
		input           string
		isValidExpected bool
	}{
		{
			explanation:     "empty passphrases are invalid",
			input:           "",
			isValidExpected: false,
		},
		{
			explanation:     "one ASCII character is valid",
			input:           "a",
			isValidExpected: true,
		},
		{
			explanation:     "one emoji is valid",
			input:           "🔒",
			isValidExpected: true,
		},
		{
			explanation:     "100 ASCII characters are valid",
			input:           strings.Repeat("a", picoshare.MaxPassphraseCodePoints),
			isValidExpected: true,
		},
		{
			explanation:     "100 emoji are valid",
			input:           strings.Repeat("🔒", picoshare.MaxPassphraseCodePoints),
			isValidExpected: true,
		},
		{
			explanation:     "101 Unicode code points are invalid",
			input:           strings.Repeat("🔒", picoshare.MaxPassphraseCodePoints+1),
			isValidExpected: false,
		},
		{
			explanation:     "arbitrary characters are preserved",
			input:           " \t\n<script>\x00&'\\é",
			isValidExpected: true,
		},
		{
			explanation:     "invalid UTF-8 is rejected",
			input:           string([]byte{0xff}),
			isValidExpected: false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			passphrase, err := picoshare.NewPassphrase(tt.input)
			isValid := err == nil
			if got, want := isValid, tt.isValidExpected; got != want {
				t.Fatalf("NewPassphrase validity=%t, want=%t", got, want)
			}
			if !isValid {
				return
			}
			if got, want := string(passphrase.Bytes()), tt.input; got != want {
				t.Errorf("passphrase=%q, want=%q", got, want)
			}
		})
	}
}
