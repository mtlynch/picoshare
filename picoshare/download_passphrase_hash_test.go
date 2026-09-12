package picoshare_test

import (
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

// knownHash is the hash of "correct horse battery staple" with the salt
// "0123456789abcdef".
const knownHash = "pbkdf2-sha256$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZg$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY"

func TestHashDownloadPassphraseRoundTrip(t *testing.T) {
	passphrase, err := picoshare.NewPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatalf("failed to create passphrase: %v", err)
	}
	hash, err := picoshare.HashDownloadPassphrase(passphrase)
	if err != nil {
		t.Fatalf("failed to hash passphrase: %v", err)
	}

	parsed, err := picoshare.ParseDownloadPassphraseHash(hash.Encoded())
	if err != nil {
		t.Fatalf("failed to parse encoded hash %q: %v", hash.Encoded(), err)
	}
	if got, want := parsed.Encoded(), hash.Encoded(); got != want {
		t.Errorf("encoded=%q, want=%q", got, want)
	}
	if !parsed.Matches(passphrase) {
		t.Error("parsed hash does not match the passphrase it was created from")
	}
}

func TestDownloadPassphraseHashMatches(t *testing.T) {
	hash, err := picoshare.ParseDownloadPassphraseHash(knownHash)
	if err != nil {
		t.Fatalf("failed to parse known hash: %v", err)
	}
	for _, tt := range []struct {
		passphrase      string
		matchesExpected bool
	}{
		{
			passphrase:      "correct horse battery staple",
			matchesExpected: true,
		},
		{
			passphrase:      "correct horse battery staple ",
			matchesExpected: false,
		},
		{
			passphrase:      "Correct horse battery staple",
			matchesExpected: false,
		},
		{
			passphrase:      "wrong passphrase",
			matchesExpected: false,
		},
	} {
		t.Run(tt.passphrase, func(t *testing.T) {
			passphrase, err := picoshare.NewPassphrase(tt.passphrase)
			if err != nil {
				t.Fatalf("failed to create passphrase: %v", err)
			}
			if got, want := hash.Matches(passphrase), tt.matchesExpected; got != want {
				t.Errorf("matches=%v, want=%v", got, want)
			}
		})
	}
}

func TestParseDownloadPassphraseHash(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		encoded     string
		errExpected error
	}{
		{
			explanation: "well-formed hash parses",
			encoded:     knownHash,
			errExpected: nil,
		},
		{
			explanation: "empty string is invalid",
			encoded:     "",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "wrong algorithm is invalid",
			encoded:     "argon2id$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZg$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "wrong version is invalid",
			encoded:     "pbkdf2-sha256$v=2$i=600000$MDEyMzQ1Njc4OWFiY2RlZg$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "wrong iteration count is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=1000$MDEyMzQ1Njc4OWFiY2RlZg$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "missing key field is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZg",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "salt that is not base64 is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=600000$not*base64$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "salt of the wrong length is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZjAx$bEpkaq0Q0Get1ft52QeKFtqD1Q+BZwqOdZOySebZSTY",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "key that is not base64 is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZg$not*base64",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
		{
			explanation: "key of the wrong length is invalid",
			encoded:     "pbkdf2-sha256$v=1$i=600000$MDEyMzQ1Njc4OWFiY2RlZg$bEpkaq0Q0Get1ft52QeKFtqD1Q",
			errExpected: picoshare.ErrInvalidDownloadPassphraseHash,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			hash, err := picoshare.ParseDownloadPassphraseHash(tt.encoded)
			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if err != nil {
				return
			}
			if got, want := hash.Encoded(), tt.encoded; got != want {
				t.Errorf("encoded=%q, want=%q", got, want)
			}
		})
	}
}
