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

	// ErrInvalidBase64 indicates that the provided base64 string is empty or malformed.
	ErrInvalidBase64 = errors.New("invalid shared secret")
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

	dk := pbkdf2.Key(key, kdf.salt, kdf.iter, kdf.keyLength, sha256.New)
	kdf.derivedKey = dk
	return kdf, nil
}

// CompareWithInput derives a key from the input and compares it with the stored derived key.
func (k *Pbkdf2KDF) CompareWithInput(inputKey []byte) bool {
	if len(inputKey) == 0 {
		return false
	}

	dk := pbkdf2.Key(inputKey, k.salt, k.iter, k.keyLength, sha256.New)
	return subtle.ConstantTimeCompare(dk, k.derivedKey) != 0
}

// CompareWithDerived compares a provided derived key with the stored derived key.
func (k *Pbkdf2KDF) CompareWithDerived(derivedKey []byte) bool {
	return subtle.ConstantTimeCompare(derivedKey, k.derivedKey) != 0
}

// GetDerivedKey returns the internally stored derived key.
func (k *Pbkdf2KDF) GetDerivedKey() []byte {
	// Return a copy to prevent external modification
	result := make([]byte, len(k.derivedKey))
	copy(result, k.derivedKey)
	return result
}

// FromBase64 decodes a base64-encoded key.
func (k *Pbkdf2KDF) FromBase64(b64encoded string) ([]byte, error) {
	if len(b64encoded) == 0 {
		return nil, ErrInvalidBase64
	}

	decoded, err := base64.StdEncoding.DecodeString(b64encoded)
	if err != nil {
		return nil, ErrInvalidBase64
	}

	return decoded, nil
}

// Compare securely compares two keys.
func (k *Pbkdf2KDF) Compare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) != 0
}
