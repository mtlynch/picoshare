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
			if got, want := passphrase.String(), tt.input; got != want {
				t.Errorf("passphrase=%q, want=%q", got, want)
			}
		})
	}
}

func TestPassphraseEqual(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		a             string
		b             string
		equalExpected bool
	}{
		{
			explanation:   "identical passphrases are equal",
			a:             "correct horse battery staple",
			b:             "correct horse battery staple",
			equalExpected: true,
		},
		{
			explanation:   "passphrases that differ by case are not equal",
			a:             "correct horse battery staple",
			b:             "Correct horse battery staple",
			equalExpected: false,
		},
		{
			explanation:   "passphrases that differ by trailing whitespace are not equal",
			a:             "correct horse battery staple",
			b:             "correct horse battery staple ",
			equalExpected: false,
		},
		{
			explanation:   "passphrases of different lengths are not equal",
			a:             "correct horse battery staple",
			b:             "correct",
			equalExpected: false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			a, err := picoshare.NewPassphrase(tt.a)
			if err != nil {
				t.Fatalf("failed to create passphrase %q: %v", tt.a, err)
			}
			b, err := picoshare.NewPassphrase(tt.b)
			if err != nil {
				t.Fatalf("failed to create passphrase %q: %v", tt.b, err)
			}
			if got, want := a.Equal(b), tt.equalExpected; got != want {
				t.Errorf("equal=%v, want=%v", got, want)
			}
		})
	}
}
