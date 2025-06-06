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
				cookieValue := k.CreateCookieValue()
				if len(cookieValue) == 0 {
					t.Errorf("cookie value is empty, expected non-empty output")
				}
			} else if k != nil {
				t.Errorf("kdf=%v, want=nil when error occurs", k)
			}
		})
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
		input       []byte
		expected    bool
	}{
		{
			description: "same key matches",
			input:       []byte("test"),
			expected:    true,
		},
		{
			description: "different key doesn't match",
			input:       []byte("different"),
			expected:    false,
		},
		{
			description: "empty key doesn't match",
			input:       []byte{},
			expected:    false,
		},
	} {
		t.Run(fmt.Sprintf("raw input: %s", tt.description), func(t *testing.T) {
			result := k.Compare(tt.input)
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

	// Get derived keys by decoding cookie values
	k1DerivedKey, err := kdf.DecodeBase64(k1Dup.CreateCookieValue())
	if err != nil {
		t.Fatalf("failed to decode k1 cookie: %v", err)
	}

	k2DerivedKey, err := kdf.DecodeBase64(k2.CreateCookieValue())
	if err != nil {
		t.Fatalf("failed to decode k2 cookie: %v", err)
	}

	for _, tt := range []struct {
		description string
		derivedKey  []byte
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
			derivedKey:  []byte{},
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

func TestCreateCookieValue(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	cookieValue := k.CreateCookieValue()
	if len(cookieValue) == 0 {
		t.Errorf("cookie value is empty, expected non-empty output")
	}

	// Test that we can decode the cookie value
	decoded, err := kdf.DecodeBase64(cookieValue)
	if err != nil {
		t.Errorf("failed to decode cookie value: %v", err)
	}

	if len(decoded) == 0 {
		t.Errorf("decoded cookie value is empty")
	}

	// Test that the decoded value works with Compare
	if !k.Compare(decoded) {
		t.Errorf("decoded cookie value doesn't match with Compare")
	}
}

func TestDecodeBase64(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	validCookieValue := k.CreateCookieValue()

	for _, tt := range []struct {
		description string
		input       string
		expectError bool
	}{
		{
			description: "accept valid base64",
			input:       validCookieValue,
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
			decoded, err := kdf.DecodeBase64(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if decoded != nil {
					t.Errorf("decoded=%v, want=nil when error occurs", decoded)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(decoded) == 0 {
					t.Errorf("decoded value is empty, expected non-empty output")
				}
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// Test the full flow: create KDF, generate cookie, decode and compare
	secret := "mysecret"
	k, err := kdf.New([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	// Test login flow
	if !k.Compare([]byte(secret)) {
		t.Errorf("login comparison failed")
	}

	if k.Compare([]byte("wrongsecret")) {
		t.Errorf("login comparison should have failed for wrong secret")
	}

	// Test cookie flow
	cookieValue := k.CreateCookieValue()
	decodedCookie, err := kdf.DecodeBase64(cookieValue)
	if err != nil {
		t.Fatalf("failed to decode cookie: %v", err)
	}

	if !k.Compare(decodedCookie) {
		t.Errorf("cookie comparison failed")
	}

	// Test that different KDF instances with same secret work
	k2, err := kdf.New([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create second KDF: %v", err)
	}

	if !k2.Compare([]byte(secret)) {
		t.Errorf("second KDF login comparison failed")
	}

	if !k2.Compare(decodedCookie) {
		t.Errorf("second KDF cookie comparison failed")
	}
}
