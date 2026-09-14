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
			input:           strings.Repeat("a", picoshare.MaxPassphraseLength),
			isValidExpected: true,
		},
		{
			explanation:     "100 emoji are valid",
			input:           strings.Repeat("🔒", picoshare.MaxPassphraseLength),
			isValidExpected: true,
		},
		{
			explanation:     "101 Unicode code points are invalid",
			input:           strings.Repeat("🔒", picoshare.MaxPassphraseLength+1),
			isValidExpected: false,
		},
		{
			explanation:     "arbitrary characters are preserved",
			input:           " \t\n<script>&'\\é",
			isValidExpected: true,
		},
		{
			explanation:     "NUL bytes are rejected",
			input:           "abc\x00def",
			isValidExpected: false,
		},
		{
			explanation:     "a lone NUL byte is rejected",
			input:           "\x00",
			isValidExpected: false,
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

func TestDownloadPassphraseEmpty(t *testing.T) {
	if got, want := (picoshare.DownloadPassphrase{}).Empty(), true; got != want {
		t.Errorf("empty=%v, want=%v", got, want)
	}

	passphrase, err := picoshare.NewDownloadPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create download passphrase: %v", err)
	}
	if got, want := passphrase.Empty(), false; got != want {
		t.Errorf("empty=%v, want=%v", got, want)
	}
}

func TestDownloadPassphraseEqual(t *testing.T) {
	emptyPassphrase := picoshare.DownloadPassphrase{}
	nonEmptyPassphrase, err := picoshare.NewDownloadPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create download passphrase: %v", err)
	}

	for _, tt := range []struct {
		explanation   string
		a             picoshare.DownloadPassphrase
		b             picoshare.DownloadPassphrase
		equalExpected bool
	}{
		{
			explanation:   "two empty passphrases are equal",
			a:             emptyPassphrase,
			b:             emptyPassphrase,
			equalExpected: true,
		},
		{
			explanation:   "an empty passphrase does not equal a non-empty passphrase",
			a:             emptyPassphrase,
			b:             nonEmptyPassphrase,
			equalExpected: false,
		},
		{
			explanation:   "a non-empty passphrase does not equal an empty passphrase",
			a:             nonEmptyPassphrase,
			b:             emptyPassphrase,
			equalExpected: false,
		},
		{
			explanation:   "identical passphrases are equal",
			a:             mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			b:             mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			equalExpected: true,
		},
		{
			explanation:   "passphrases that differ by case are not equal",
			a:             mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			b:             mustCreateDownloadPassphrase(t, "Correct horse battery staple"),
			equalExpected: false,
		},
		{
			explanation:   "passphrases that differ by trailing whitespace are not equal",
			a:             mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			b:             mustCreateDownloadPassphrase(t, "correct horse battery staple "),
			equalExpected: false,
		},
		{
			explanation:   "passphrases of different lengths are not equal",
			a:             mustCreateDownloadPassphrase(t, "correct horse battery staple"),
			b:             mustCreateDownloadPassphrase(t, "correct"),
			equalExpected: false,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := tt.a.Equal(tt.b), tt.equalExpected; got != want {
				t.Errorf("equal=%v, want=%v", got, want)
			}
		})
	}
}

func mustCreateDownloadPassphrase(t *testing.T, raw string) picoshare.DownloadPassphrase {
	t.Helper()

	passphrase, err := picoshare.NewDownloadPassphrase(raw)
	if err != nil {
		t.Fatalf("failed to create download passphrase %q: %v", raw, err)
	}
	return passphrase
}
