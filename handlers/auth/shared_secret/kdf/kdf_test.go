package kdf_test

import (
	"fmt"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

func TestNewDeriver(t *testing.T) {
	deriver := kdf.NewDeriver()
	if deriver == nil {
		t.Fatal("NewDeriver() returned nil")
	}
}

func TestDerive(t *testing.T) {
	deriver := kdf.NewDeriver()

	for _, tt := range []struct {
		description string
		input       string
		expectError bool
		err         error
	}{
		{
			description: "accept valid secret",
			input:       "mysecret",
			expectError: false,
			err:         nil,
		},
		{
			description: "reject empty secret",
			input:       "",
			expectError: true,
			err:         kdf.ErrInvalidSecret,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			_, err := deriver.Derive(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
		})
	}
}

func TestKeyComparison(t *testing.T) {
	deriver := kdf.NewDeriver()

	key1, err := deriver.Derive("test")
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	// Test that same secret creates matching keys
	key2, err := deriver.Derive("test")
	if err != nil {
		t.Fatalf("failed to derive second key: %v", err)
	}

	if !key1.Equal(key2) {
		t.Errorf("keys with same secret should match")
	}

	// Test with different secrets
	key3, err := deriver.Derive("different-secret")
	if err != nil {
		t.Fatalf("failed to derive key3: %v", err)
	}

	if key1.Equal(key3) {
		t.Errorf("keys with different secrets should not match")
	}

	// Test comparison with empty struct.
	if key1.Equal(kdf.DerivedKey{}) {
		t.Errorf("comparison with nil should return false")
	}
}

func TestDeserializeKey(t *testing.T) {
	deriver := kdf.NewDeriver()
	key, err := deriver.Derive("test")
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	// Get a valid base64 representation
	serialized := key.Serialize()

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
			deserializedKey, err := kdf.DeserializeKey(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				//if deserializedKey != nil {
				//	t.Errorf("deserializedKey=%v, want=nil when error occurs", deserializedKey)
				//}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Test that the deserialized key works with Equal.
				if !key.Equal(deserializedKey) {
					t.Errorf("deserialized key doesn't match original")
				}
			}
		})
	}
}

func TestSerializeDeserialize(t *testing.T) {
	deriver := kdf.NewDeriver()
	key, err := deriver.Derive("test")
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	// Test serialization
	serialized := key.Serialize()
	if len(serialized) == 0 {
		t.Errorf("serialized value is empty")
	}

	// Test deserialization
	deserializedKey, err := kdf.DeserializeKey(serialized)
	if err != nil {
		t.Errorf("failed to deserialize: %v", err)
	}

	if !key.Equal(deserializedKey) {
		t.Errorf("deserialized key doesn't match original")
	}

	// Test round-trip: serialize -> deserialize -> serialize
	serialized2 := deserializedKey.Serialize()
	if serialized != serialized2 {
		t.Errorf("round-trip serialization failed: %q != %q", serialized, serialized2)
	}
}

func TestCompare(t *testing.T) {
	deriver := kdf.NewDeriver()

	// Test with different secrets
	for _, tt := range []struct {
		description string
		secret1     string
		secret2     string
		expected    bool
	}{
		{
			description: "same secret matches",
			secret1:     "test",
			secret2:     "test",
			expected:    true,
		},
		{
			description: "different secrets don't match",
			secret1:     "test",
			secret2:     "different",
			expected:    false,
		},
		{
			description: "empty secret doesn't match non-empty",
			secret1:     "test",
			secret2:     "",
			expected:    false,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			key1, err := deriver.Derive(tt.secret1)
			if err != nil {
				t.Fatalf("failed to derive first key: %v", err)
			}

			if tt.secret2 == "" {
				// Test with uninitialized struct for empty secret case.
				if got, want := key1.Equal(kdf.DerivedKey{}), tt.expected; got != want {
					t.Errorf("result=%v, want=%v", got, want)
				}
			} else {
				key2, err := deriver.Derive(tt.secret2)
				if err != nil {
					t.Fatalf("failed to derive second key: %v", err)
				}

				if got, want := key1.Equal(key2), tt.expected; got != want {
					t.Errorf("result=%v, want=%v", got, want)
				}
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// Test the full flow: create deriver, derive keys, compare and serialize
	deriver := kdf.NewDeriver()
	secret := "mysecret"

	serverKey, err := deriver.Derive(secret)
	if err != nil {
		t.Fatalf("failed to derive server key: %v", err)
	}

	// Test login flow
	userKey, err := deriver.Derive(secret)
	if err != nil {
		t.Fatalf("failed to derive user key: %v", err)
	}

	if !serverKey.Equal(userKey) {
		t.Errorf("login comparison failed")
	}

	wrongUserKey, err := deriver.Derive("wrongsecret")
	if err != nil {
		t.Fatalf("failed to derive wrong user key: %v", err)
	}

	if serverKey.Equal(wrongUserKey) {
		t.Errorf("login comparison should have failed for wrong secret")
	}

	// Test cookie flow
	serializedValue := serverKey.Serialize()
	cookieKey, err := kdf.DeserializeKey(serializedValue)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if !serverKey.Equal(cookieKey) {
		t.Errorf("cookie comparison failed")
	}

	// Test that different deriver instances with same secret work
	deriver2 := kdf.NewDeriver()
	serverKey2, err := deriver2.Derive(secret)
	if err != nil {
		t.Fatalf("failed to derive second server key: %v", err)
	}

	if !serverKey2.Equal(userKey) {
		t.Errorf("second deriver login comparison failed")
	}

	if !serverKey2.Equal(cookieKey) {
		t.Errorf("second deriver cookie comparison failed")
	}
}
