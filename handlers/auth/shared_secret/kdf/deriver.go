package kdf

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrInvalidSecret indicates that the provided secret is empty or invalid.
	ErrInvalidSecret = errors.New("invalid shared secret")
	// ErrInvalidSerialization indicates that the serialized data is invalid.
	ErrInvalidSerialization = errors.New("invalid serialized key data")
)

// DeriveKeyFromSecret creates a derived key from the provided secret string
// using PBKDF2 with hardcoded parameters.
func DeriveKeyFromSecret(secret string) (DerivedKey, error) {
	if secret == "" {
		return DerivedKey{}, ErrInvalidSecret
	}

	// These would be insecure values for storing a database of user credentials,
	// but we're only storing a single password, so it's not important to have
	// random salt or high iteration rounds.
	salt := []byte{1, 2, 3, 4}
	iter := 100
	keyLength := 32

	keyData := pbkdf2.Key([]byte(secret), salt, iter, keyLength, sha256.New)
	return DerivedKey{data: keyData}, nil
}
