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
	// ErrInvalidSerialization indicates that the serialized data is invalid.
	ErrInvalidSerialization = errors.New("invalid serialized key data")
)

// Pbkdf2KDF represents a key derivation function using PBKDF2.
type Pbkdf2KDF struct {
	salt       []byte
	iter       int
	keyLength  int
	derivedKey []byte
}

// New creates a new KDF instance by deriving a key from the raw secret string.
func New(rawSecret string) (*Pbkdf2KDF, error) {
	if rawSecret == "" {
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

	kdf.derivedKey = pbkdf2.Key([]byte(rawSecret), kdf.salt, kdf.iter, kdf.keyLength, sha256.New)
	return kdf, nil
}

// Deserialize creates a KDF instance from a base64-encoded derived key.
func Deserialize(base64Data string) (*Pbkdf2KDF, error) {
	if base64Data == "" {
		return nil, ErrInvalidSerialization
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, ErrInvalidSerialization
	}

	kdf := &Pbkdf2KDF{
		salt:       []byte{1, 2, 3, 4},
		iter:       100,
		keyLength:  32,
		derivedKey: decoded,
	}

	return kdf, nil
}

// Compare performs constant-time comparison between this KDF and another KDF.
func (k *Pbkdf2KDF) Compare(other *Pbkdf2KDF) bool {
	if other == nil || len(other.derivedKey) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(k.derivedKey, other.derivedKey) != 0
}

// Serialize returns the base64-encoded representation of the derived key.
func (k *Pbkdf2KDF) Serialize() string {
	return base64.StdEncoding.EncodeToString(k.derivedKey)
}
