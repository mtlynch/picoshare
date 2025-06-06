package kdf_test

import (
	"fmt"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

func TestNew(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		expectError bool
		err         error
	}{
		{
			description: "accept valid key",
			input:       "mysecret",
			expectError: false,
			err:         nil,
		},
		{
			description: "reject empty key",
			input:       "",
			expectError: true,
			err:         kdf.ErrInvalidKey,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
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

func TestKDFComparison(t *testing.T) {
	k1, err := kdf.New("test")
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test that same secret creates matching KDFs
	k2, err := kdf.New("test")
	if err != nil {
		t.Fatalf("failed to create second KDF: %v", err)
	}

	if !k1.Compare(k2) {
		t.Errorf("KDFs with same secret should match")
	}

	// Test that different secrets don't match
	k3, err := kdf.New("different")
	if err != nil {
		t.Fatalf("failed to create third KDF: %v", err)
	}

	if k1.Compare(k3) {
		t.Errorf("KDFs with different secrets should not match")
	}

	// Test comparison with nil
	if k1.Compare(nil) {
		t.Errorf("comparison with nil should return false")
	}
}

func TestDeserialize(t *testing.T) {
	k, err := kdf.New("test")
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
			deserializedKDF, err := kdf.Deserialize(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if deserializedKDF != nil {
					t.Errorf("deserializedKDF=%v, want=nil when error occurs", deserializedKDF)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if deserializedKDF == nil {
					t.Errorf("deserialized KDF is nil, expected non-nil output")
				}

				// Test that the deserialized KDF works with Compare
				if !k.Compare(deserializedKDF) {
					t.Errorf("deserialized KDF doesn't match original")
				}
			}
		})
	}
}

func TestSerializeDeserialize(t *testing.T) {
	k, err := kdf.New("test")
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test serialization
	serialized := k.Serialize()
	if len(serialized) == 0 {
		t.Errorf("serialized value is empty")
	}

	// Test deserialization
	deserializedKDF, err := kdf.Deserialize(serialized)
	if err != nil {
		t.Errorf("failed to deserialize: %v", err)
	}

	if !k.Compare(deserializedKDF) {
		t.Errorf("deserialized KDF doesn't match original")
	}

	// Test round-trip: serialize -> deserialize -> serialize
	serialized2 := deserializedKDF.Serialize()
	if serialized != serialized2 {
		t.Errorf("round-trip serialization failed: %q != %q", serialized, serialized2)
	}
}

func TestCompare(t *testing.T) {
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
			k1, err := kdf.New(tt.secret1)
			if err != nil {
				t.Fatalf("failed to create first KDF: %v", err)
			}

			if tt.secret2 == "" {
				// Test with nil for empty secret case
				if got, want := k1.Compare(nil), tt.expected; got != want {
					t.Errorf("result=%v, want=%v", got, want)
				}
			} else {
				k2, err := kdf.New(tt.secret2)
				if err != nil {
					t.Fatalf("failed to create second KDF: %v", err)
				}

				if got, want := k1.Compare(k2), tt.expected; got != want {
					t.Errorf("result=%v, want=%v", got, want)
				}
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// Test the full flow: create KDF, generate serialized value, deserialize and compare
	secret := "mysecret"
	serverKDF, err := kdf.New(secret)
	if err != nil {
		t.Fatalf("failed to create server KDF: %v", err)
	}

	// Test login flow
	userKDF, err := kdf.New(secret)
	if err != nil {
		t.Fatalf("failed to create user KDF: %v", err)
	}

	if !serverKDF.Compare(userKDF) {
		t.Errorf("login comparison failed")
	}

	wrongUserKDF, err := kdf.New("wrongsecret")
	if err != nil {
		t.Fatalf("failed to create wrong user KDF: %v", err)
	}

	if serverKDF.Compare(wrongUserKDF) {
		t.Errorf("login comparison should have failed for wrong secret")
	}

	// Test cookie flow
	serializedValue := serverKDF.Serialize()
	cookieKDF, err := kdf.Deserialize(serializedValue)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if !serverKDF.Compare(cookieKDF) {
		t.Errorf("cookie comparison failed")
	}

	// Test that different server KDF instances with same secret work
	serverKDF2, err := kdf.New(secret)
	if err != nil {
		t.Fatalf("failed to create second server KDF: %v", err)
	}

	if !serverKDF2.Compare(userKDF) {
		t.Errorf("second server KDF login comparison failed")
	}

	if !serverKDF2.Compare(cookieKDF) {
		t.Errorf("second server KDF cookie comparison failed")
	}
}
