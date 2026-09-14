package picoshare

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxPassphraseLength = 100

var ErrInvalidPassphrase = fmt.Errorf("passphrase must contain between 1 and %d Unicode code points", MaxPassphraseLength)

// Passphrase is a validated plaintext passphrase.
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
	if count := utf8.RuneCountInString(raw); count < 1 || count > MaxPassphraseLength {
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
