package picoshare

import (
	"fmt"
	"unicode/utf8"
)

const MaxPassphraseCodePoints = 100

var ErrInvalidPassphrase = fmt.Errorf("passphrase must contain between 1 and %d Unicode code points", MaxPassphraseCodePoints)

// Passphrase is a validated plaintext passphrase.
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

// Bytes returns the exact UTF-8 bytes supplied when constructing the passphrase.
func (p Passphrase) Bytes() []byte {
	if p.value == "" {
		panic("cannot access an uninitialized passphrase")
	}
	return []byte(p.value)
}
