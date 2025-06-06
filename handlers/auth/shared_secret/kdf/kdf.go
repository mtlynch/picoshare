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
	salt      []byte
	iter      int
	keyLength int
}

// New creates a new KDF instance with default parameters.
func New() *Pbkdf2KDF {
	return &Pbkdf2KDF{
		// These would be insecure values for storing a database of user credentials,
		// but we're only storing a single password, so it's not important to have
		// random salt or high iteration rounds.
		salt:      []byte{1, 2, 3, 4},
		iter:      100,
		keyLength: 32,
	}
}

// DeriveFromKey derives a key using PBKDF2.
func (k *Pbkdf2KDF) DeriveFromKey(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrInvalidKey
	}

	dk := pbkdf2.Key(key, k.salt, k.iter, k.keyLength, sha256.New)
	return dk, nil
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
