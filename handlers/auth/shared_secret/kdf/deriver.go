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

// Pbkdf2Deriver configures and performs PBKDF2 key derivation.
type Pbkdf2Deriver struct {
	salt      []byte
	iter      int
	keyLength int
}

// NewDeriver creates a new PBKDF2 key deriver with default parameters.
func NewDeriver() *Pbkdf2Deriver {
	return &Pbkdf2Deriver{
		// These would be insecure values for storing a database of user credentials,
		// but we're only storing a single password, so it's not important to have
		// random salt or high iteration rounds.
		salt:      []byte{1, 2, 3, 4},
		iter:      100,
		keyLength: 32,
	}
}

// Derive creates a derived key from the provided secret string.
func (d *Pbkdf2Deriver) Derive(secret string) (*DerivedKey, error) {
	if secret == "" {
		return nil, ErrInvalidSecret
	}

	keyData := pbkdf2.Key([]byte(secret), d.salt, d.iter, d.keyLength, sha256.New)
	return &DerivedKey{data: keyData}, nil
}
