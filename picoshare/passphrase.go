package picoshare

import (
	"crypto/subtle"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode/utf8"
)

const MaxPassphraseCodePoints = 100

var ErrInvalidPassphrase = fmt.Errorf("passphrase must contain between 1 and %d Unicode code points", MaxPassphraseCodePoints)

// Passphrase is a validated plaintext passphrase. The zero value is the empty
// passphrase, which represents the absence of a passphrase.
type Passphrase struct {
	value string
}

// NewPassphrase constructs a passphrase from user-provided text.
func NewPassphrase(raw string) (Passphrase, error) {
	if !utf8.ValidString(raw) {
		return Passphrase{}, fmt.Errorf("%w: invalid UTF-8", ErrInvalidPassphrase)
	}
	// UTF-8 strings support nul bytes, but in practice, there's almost certainly
	// something fishy going on if a passphrase contains a nul byte, so disallow
	// it.
	if strings.ContainsRune(raw, 0) {
		return Passphrase{}, fmt.Errorf("%w: NUL bytes are not allowed", ErrInvalidPassphrase)
	}
	if count := utf8.RuneCountInString(raw); count < 1 || count > MaxPassphraseCodePoints {
		return Passphrase{}, ErrInvalidPassphrase
	}
	return Passphrase{value: raw}, nil
}

// Empty reports whether the passphrase is the empty passphrase.
func (p Passphrase) Empty() bool {
	return p.value == ""
}

// String returns the exact text supplied when constructing the passphrase, or
// an empty string for the empty passphrase.
func (p Passphrase) String() string {
	return p.value
}

// DownloadPassphrase is a passphrase that protects downloads of an entry.
// PicoShare stores download passphrases in plaintext, so it is safe to compare
// them directly. The zero value is the empty download passphrase, which
// represents the absence of a passphrase.
type DownloadPassphrase struct {
	passphrase Passphrase
}

// NewDownloadPassphrase constructs a download passphrase from user-provided
// text.
func NewDownloadPassphrase(raw string) (DownloadPassphrase, error) {
	passphrase, err := NewPassphrase(raw)
	if err != nil {
		return DownloadPassphrase{}, err
	}
	return DownloadPassphrase{passphrase: passphrase}, nil
}

// Empty reports whether the download passphrase is the empty passphrase.
func (p DownloadPassphrase) Empty() bool {
	return p.passphrase.Empty()
}

// String returns the exact text of the download passphrase, or an empty string
// for the empty download passphrase.
func (p DownloadPassphrase) String() string {
	return p.passphrase.String()
}

// Equal performs constant-time comparison between this download passphrase and
// another download passphrase. Two empty download passphrases are equal.
func (p DownloadPassphrase) Equal(other DownloadPassphrase) bool {
	// subtle.ConstantTimeCompare returns early when its inputs differ in
	// length, which would leak the length of the passphrase through timing.
	// Hashing both values first produces fixed-length inputs so that the
	// comparison always takes the same amount of time.
	pHash := fnv.New64a()
	_, _ = pHash.Write([]byte(p.String()))
	otherHash := fnv.New64a()
	_, _ = otherHash.Write([]byte(other.String()))
	return subtle.ConstantTimeCompare(pHash.Sum(nil), otherHash.Sum(nil)) == 1
}
