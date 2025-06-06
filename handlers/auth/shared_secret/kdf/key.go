package kdf

import (
	"crypto/subtle"
	"encoding/base64"
)

// DerivedKey represents key material derived from a key derivation function.
type DerivedKey struct {
	data []byte
}

// Equal performs constant-time comparison between this key and another key.
func (k DerivedKey) Equal(other DerivedKey) bool {
	return subtle.ConstantTimeCompare(k.data, other.data) != 0
}

// Serialize returns the base64-encoded representation of the derived key.
func (k DerivedKey) Serialize() string {
	return base64.StdEncoding.EncodeToString(k.data)
}

// DeserializeKey creates a DerivedKey from a base64-encoded string.
func DeserializeKey(base64Data string) (DerivedKey, error) {
	if base64Data == "" {
		return DerivedKey{}, ErrInvalidSerialization
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return DerivedKey{}, ErrInvalidSerialization
	}

	return DerivedKey{data: decoded}, nil
}
