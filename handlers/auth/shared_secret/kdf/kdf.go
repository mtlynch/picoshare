package kdf

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrInvalidKey indicates that the provided key is empty or invalid.
	ErrInvalidKey = errors.New("invalid shared secret key")
)

type Pbkdf2KDF struct {
	salt       []byte
	iter       int
	keyLength  int
	derivedKey []byte
}

// New creates a new KDF instance with a key that is immediately derived and stored internally.
func New(key []byte) (*Pbkdf2KDF, error) {
	if len(key) == 0 {
		return nil, ErrInvalidKey
	}

	kdf := &Pbkdf2KDF{
		// These would be insecure values for storing a database of user credentials,
		// but we're only storing a single password, so it's not important to have
		// random salt or high iteration rounds.
		salt:      []byte{1, 2, 3, 4},
		iter:      100,
		keyLength: 32,
	}

	kdf.derivedKey = pbkdf2.Key(key, kdf.salt, kdf.iter, kdf.keyLength, sha256.New)
	return kdf, nil
}

// Compare handles both raw input and base64-decoded cookie values.
// It automatically detects the input type and performs the appropriate comparison.
func (k *Pbkdf2KDF) Compare(input []byte) bool {
	if len(input) == 0 {
		return false
	}

	// First try comparing as derived key (for cookie validation)
	// Derived keys have a specific length matching our key length
	if len(input) == k.keyLength {
		if subtle.ConstantTimeCompare(input, k.derivedKey) != 0 {
			return true
		}
	}

	// Then try deriving from raw input (for login)
	derived := pbkdf2.Key(input, k.salt, k.iter, k.keyLength, sha256.New)
	return subtle.ConstantTimeCompare(derived, k.derivedKey) != 0
}

// CreateCookieValue generates base64-encoded derived key for cookies.
func (k *Pbkdf2KDF) CreateCookieValue() string {
	return base64.StdEncoding.EncodeToString(k.derivedKey)
}

// DecodeBase64 is a package-level utility function for decoding base64 strings.
func DecodeBase64(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("empty base64 string")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("invalid base64 string")
	}
	return decoded, nil
}
