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

// RawKey represents a user-provided password or secret string.
type RawKey struct {
	value string
}

// DerivedKey represents a cryptographically derived key.
type DerivedKey struct {
	value []byte
}

// NewRawKey creates a new RawKey from a string input.
func NewRawKey(input string) RawKey {
	return RawKey{value: input}
}

// String returns the raw string value.
func (r RawKey) String() string {
	return r.value
}

// Derive converts a RawKey to a DerivedKey using the provided KDF.
func (r RawKey) Derive(k *Pbkdf2KDF) DerivedKey {
	derived := pbkdf2.Key([]byte(r.value), k.salt, k.iter, k.keyLength, sha256.New)
	return DerivedKey{value: derived}
}

// NewDerivedKeyFromBase64 creates a DerivedKey from a base64-encoded string.
func NewDerivedKeyFromBase64(encoded string) (DerivedKey, error) {
	if encoded == "" {
		return DerivedKey{}, ErrInvalidSerialization
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return DerivedKey{}, ErrInvalidSerialization
	}
	return DerivedKey{value: decoded}, nil
}

// Bytes returns the raw byte slice value.
func (d DerivedKey) Bytes() []byte {
	return d.value
}

// ToBase64 returns the base64-encoded representation of the derived key.
func (d DerivedKey) ToBase64() string {
	return base64.StdEncoding.EncodeToString(d.value)
}

// KDF defines the interface for key derivation operations.
type KDF interface {
	Compare(key DerivedKey) bool
	Serialize() string
	Deserialize(s string) (DerivedKey, error)
}

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

// Compare performs constant-time comparison between the stored derived key and the input derived key.
func (k *Pbkdf2KDF) Compare(input DerivedKey) bool {
	if len(input.value) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(input.value, k.derivedKey) != 0
}

// Serialize returns the base64-encoded representation of the internal derived key.
func (k *Pbkdf2KDF) Serialize() string {
	return base64.StdEncoding.EncodeToString(k.derivedKey)
}

// Deserialize creates a DerivedKey from a base64-encoded string.
func (k *Pbkdf2KDF) Deserialize(s string) (DerivedKey, error) {
	return NewDerivedKeyFromBase64(s)
}
