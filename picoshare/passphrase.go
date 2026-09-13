package picoshare

import (
	"crypto/md5"
	"crypto/subtle"
	"fmt"
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
	// MD5 is not a secure hash, but that doesn't matter here. We never store
	// or transmit the digests, and we only use them to compare the plaintext
	// values already in memory. An attacker can't submit a digest directly,
	// so MD5's weakness to collision attacks is irrelevant: forging a
	// collision would require finding a second preimage of the passphrase,
	// which is no easier than guessing the passphrase itself.
	pHash := md5.Sum([]byte(p.value))
	otherHash := md5.Sum([]byte(other.value))
	return subtle.ConstantTimeCompare(pHash[:], otherHash[:]) == 1
}
