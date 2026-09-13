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
	// SQLite's length() function stops counting at the first NUL byte in a
	// TEXT value, so a passphrase containing NUL would fail the database's
	// CHECK constraint even though it satisfies the code point limits below.
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

// Equal performs constant-time comparison between this passphrase and another
// passphrase. Two empty passphrases are equal.
func (p Passphrase) Equal(other Passphrase) bool {
	// subtle.ConstantTimeCompare returns early when its inputs differ in
	// length, which would leak the length of the passphrase through timing.
	// Hashing both values first produces fixed-length inputs so that the
	// comparison always takes the same amount of time.
	//
	pHash := fnv.New64a()
	_, _ = pHash.Write([]byte(p.value))
	otherHash := fnv.New64a()
	_, _ = otherHash.Write([]byte(other.value))
	return subtle.ConstantTimeCompare(pHash.Sum(nil), otherHash.Sum(nil)) == 1
}
