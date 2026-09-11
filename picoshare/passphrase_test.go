package picoshare_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestNewPassphrase(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		input       string
		errExpected bool
	}{
		{
			explanation: "empty passphrases are invalid",
			input:       "",
			errExpected: true,
		},
		{
			explanation: "one ASCII character is valid",
			input:       "a",
		},
		{
			explanation: "one emoji is valid",
			input:       "🔒",
		},
		{
			explanation: "100 ASCII characters are valid",
			input:       strings.Repeat("a", picoshare.MaxPassphraseCodePoints),
		},
		{
			explanation: "100 emoji are valid",
			input:       strings.Repeat("🔒", picoshare.MaxPassphraseCodePoints),
		},
		{
			explanation: "101 Unicode code points are invalid",
			input:       strings.Repeat("🔒", picoshare.MaxPassphraseCodePoints+1),
			errExpected: true,
		},
		{
			explanation: "arbitrary characters are preserved",
			input:       " \t\n<script>\x00&'\\é",
		},
		{
			explanation: "invalid UTF-8 is rejected",
			input:       string([]byte{0xff}),
			errExpected: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			passphrase, err := picoshare.NewPassphrase(tt.input)
			if tt.errExpected {
				if err == nil {
					t.Fatal("NewPassphrase returned nil error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPassphrase failed: %v", err)
			}
			if got, want := string(passphrase.Bytes()), tt.input; got != want {
				t.Errorf("passphrase=%q, want=%q", got, want)
			}
		})
	}
}

func TestPassphraseBytesPanicsWhenUninitialized(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("Bytes on an uninitialized passphrase did not panic")
		}
	}()

	_ = picoshare.Passphrase{}.Bytes()
}
