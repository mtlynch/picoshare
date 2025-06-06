package kdf_test

import (
	"fmt"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

func TestNew(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       []byte
		expectError bool
		err         error
	}{
		{
			description: "accept valid key",
			input:       []byte("mysecret"),
			expectError: false,
			err:         nil,
		},
		{
			description: "reject empty key",
			input:       []byte{},
			expectError: true,
			err:         kdf.ErrInvalidKey,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, string(tt.input)), func(t *testing.T) {
			k, err := kdf.New(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if !tt.expectError {
				if k == nil {
					t.Errorf("kdf is nil, expected non-nil instance")
				}
				serialized := k.Serialize()
				if len(serialized) == 0 {
					t.Errorf("serialized value is empty, expected non-empty output")
				}
			} else if k != nil {
				t.Errorf("kdf=%v, want=nil when error occurs", k)
			}
		})
	}
}

func TestRawKeyAndDerivedKey(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test RawKey creation and derivation
	rawKey := kdf.NewRawKey("test")
	if rawKey.String() != "test" {
		t.Errorf("RawKey.String()=%q, want=%q", rawKey.String(), "test")
	}

	derivedKey := rawKey.Derive(k)
	if len(derivedKey.Bytes()) == 0 {
		t.Errorf("derived key is empty")
	}

	// Test that derived key matches with Compare
	if !k.Compare(derivedKey) {
		t.Errorf("derived key doesn't match with Compare")
	}

	// Test different raw key doesn't match
	differentRawKey := kdf.NewRawKey("different")
	differentDerivedKey := differentRawKey.Derive(k)
	if k.Compare(differentDerivedKey) {
		t.Errorf("different derived key should not match")
	}
}

func TestDerivedKeyFromBase64(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Get a valid base64 representation
	serialized := k.Serialize()

	for _, tt := range []struct {
		description string
		input       string
		expectError bool
	}{
		{
			description: "accept valid base64",
			input:       serialized,
			expectError: false,
		},
		{
			description: "reject empty string",
			input:       "",
			expectError: true,
		},
		{
			description: "reject invalid base64",
			input:       "not-base64!",
			expectError: true,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			derivedKey, err := kdf.NewDerivedKeyFromBase64(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if len(derivedKey.Bytes()) != 0 {
					t.Errorf("derivedKey.Bytes()=%v, want=empty when error occurs", derivedKey.Bytes())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(derivedKey.Bytes()) == 0 {
					t.Errorf("derived key bytes are empty, expected non-empty output")
				}

				// Test that the derived key works with Compare
				if !k.Compare(derivedKey) {
					t.Errorf("derived key from base64 doesn't match with Compare")
				}
			}
		})
	}
}

func TestSerializeDeserialize(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test serialization
	serialized := k.Serialize()
	if len(serialized) == 0 {
		t.Errorf("serialized value is empty")
	}

	// Test deserialization
	derivedKey, err := k.Deserialize(serialized)
	if err != nil {
		t.Errorf("failed to deserialize: %v", err)
	}

	if !k.Compare(derivedKey) {
		t.Errorf("deserialized key doesn't match with Compare")
	}

	// Test round-trip: serialize -> deserialize -> serialize
	derivedKey2, err := k.Deserialize(serialized)
	if err != nil {
		t.Errorf("failed to deserialize again: %v", err)
	}

	serialized2 := derivedKey2.ToBase64()
	if serialized != serialized2 {
		t.Errorf("round-trip serialization failed: %q != %q", serialized, serialized2)
	}
}

func TestCompare(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test with raw input (login scenario)
	for _, tt := range []struct {
		description string
		input       string
		expected    bool
	}{
		{
			description: "same key matches",
			input:       "test",
			expected:    true,
		},
		{
			description: "different key doesn't match",
			input:       "different",
			expected:    false,
		},
		{
			description: "empty key doesn't match",
			input:       "",
			expected:    false,
		},
	} {
		t.Run(fmt.Sprintf("raw input: %s", tt.description), func(t *testing.T) {
			rawKey := kdf.NewRawKey(tt.input)
			derivedKey := rawKey.Derive(k)
			result := k.Compare(derivedKey)
			if got, want := result, tt.expected; got != want {
				t.Errorf("result=%v, want=%v", got, want)
			}
		})
	}

	// Test with derived key input (cookie scenario)
	k1, err := kdf.New([]byte("test1"))
	if err != nil {
		t.Fatalf("failed to create KDF1: %v", err)
	}

	k2, err := kdf.New([]byte("test2"))
	if err != nil {
		t.Fatalf("failed to create KDF2: %v", err)
	}

	k1Dup, err := kdf.New([]byte("test1"))
	if err != nil {
		t.Fatalf("failed to create KDF1 duplicate: %v", err)
	}

	// Get derived keys by deserializing serialized values
	k1DerivedKey, err := k1Dup.Deserialize(k1Dup.Serialize())
	if err != nil {
		t.Fatalf("failed to deserialize k1: %v", err)
	}

	k2DerivedKey, err := k2.Deserialize(k2.Serialize())
	if err != nil {
		t.Fatalf("failed to deserialize k2: %v", err)
	}

	emptyDerivedKey := kdf.DerivedKey{}

	for _, tt := range []struct {
		description string
		derivedKey  kdf.DerivedKey
		expected    bool
	}{
		{
			description: "same derived key matches",
			derivedKey:  k1DerivedKey,
			expected:    true,
		},
		{
			description: "different derived key doesn't match",
			derivedKey:  k2DerivedKey,
			expected:    false,
		},
		{
			description: "empty key doesn't match",
			derivedKey:  emptyDerivedKey,
			expected:    false,
		},
	} {
		t.Run(fmt.Sprintf("derived key: %s", tt.description), func(t *testing.T) {
			result := k1.Compare(tt.derivedKey)
			if got, want := result, tt.expected; got != want {
				t.Errorf("result=%v, want=%v", got, want)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// Test the full flow: create KDF, generate serialized value, deserialize and compare
	secret := "mysecret"
	k, err := kdf.New([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test login flow
	rawKey := kdf.NewRawKey(secret)
	derivedKey := rawKey.Derive(k)
	if !k.Compare(derivedKey) {
		t.Errorf("login comparison failed")
	}

	wrongRawKey := kdf.NewRawKey("wrongsecret")
	wrongDerivedKey := wrongRawKey.Derive(k)
	if k.Compare(wrongDerivedKey) {
		t.Errorf("login comparison should have failed for wrong secret")
	}

	// Test cookie flow
	serializedValue := k.Serialize()
	deserializedKey, err := k.Deserialize(serializedValue)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if !k.Compare(deserializedKey) {
		t.Errorf("cookie comparison failed")
	}

	// Test that different KDF instances with same secret work
	k2, err := kdf.New([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create second KDF: %v", err)
	}

	rawKey2 := kdf.NewRawKey(secret)
	derivedKey2 := rawKey2.Derive(k2)
	if !k2.Compare(derivedKey2) {
		t.Errorf("second KDF login comparison failed")
	}

	if !k2.Compare(deserializedKey) {
		t.Errorf("second KDF cookie comparison failed")
	}
}
