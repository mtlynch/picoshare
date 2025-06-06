package kdf_test

import (
	"encoding/base64"
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
				derivedKey := k.GetDerivedKey()
				if len(derivedKey) == 0 {
					t.Errorf("derived key is empty, expected non-empty output")
				}
			} else if k != nil {
				t.Errorf("kdf=%v, want=nil when error occurs", k)
			}
		})
	}
}

func TestCompareWithInput(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

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
		t.Run(tt.description, func(t *testing.T) {
			result := k.CompareWithInput(tt.input)
			if got, want := result, tt.expected; got != want {
				t.Errorf("result=%v, want=%v", got, want)
			}
		})
	}
}

func TestCompareWithDerived(t *testing.T) {
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

	for _, tt := range []struct {
		description string
		derivedKey  []byte
		expected    bool
	}{
		{
			description: "same derived key matches",
			derivedKey:  k1Dup.GetDerivedKey(),
			expected:    true,
		},
		{
			description: "different derived key doesn't match",
			derivedKey:  k2.GetDerivedKey(),
			expected:    false,
		},
		{
			description: "empty key doesn't match",
			derivedKey:  []byte{},
			expected:    false,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			result := k1.CompareWithDerived(tt.derivedKey)
			if got, want := result, tt.expected; got != want {
				t.Errorf("result=%v, want=%v", got, want)
			}
		})
	}
}

func TestGetDerivedKey(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	derivedKey := k.GetDerivedKey()
	if len(derivedKey) == 0 {
		t.Errorf("derived key is empty, expected non-empty output")
	}

	// Test that we get a copy, not the original
	derivedKey[0] = 0xFF
	derivedKey2 := k.GetDerivedKey()
	if derivedKey[0] == derivedKey2[0] {
		t.Errorf("GetDerivedKey returned the same slice, expected a copy")
	}
}

func TestFromBase64(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	validKey := k.GetDerivedKey()
	validBase64 := base64.StdEncoding.EncodeToString(validKey)

	for _, tt := range []struct {
		description string
		input       string
		output      []byte
		err         error
	}{
		{
			description: "accept valid base64",
			input:       validBase64,
			output:      validKey,
			err:         nil,
		},
		{
			description: "reject empty string",
			input:       "",
			output:      nil,
			err:         kdf.ErrInvalidBase64,
		},
		{
			description: "reject invalid base64",
			input:       "not-base64!",
			output:      nil,
			err:         kdf.ErrInvalidBase64,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			key, err := k.FromBase64(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if err == nil {
				if len(key) == 0 {
					t.Errorf("key is empty, expected non-empty output")
				}
				if tt.output != nil && tt.description == "accept valid base64" {
					if got, want := key, tt.output; base64.StdEncoding.EncodeToString(got) != base64.StdEncoding.EncodeToString(want) {
						t.Errorf("key=%v, want=%v", base64.StdEncoding.EncodeToString(got), base64.StdEncoding.EncodeToString(want))
					}
				}
			} else if key != nil {
				t.Errorf("key=%v, want=nil when error occurs", key)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	k, err := kdf.New([]byte("test"))
	if err != nil {
		t.Fatalf("failed to create KDF: %v", err)
	}

	key1 := []byte("test1")
	key2 := []byte("test2")
	key1Dup := []byte("test1")

	for _, tt := range []struct {
		description string
		a           []byte
		b           []byte
		output      bool
	}{
		{
			description: "same keys match",
			a:           key1,
			b:           key1Dup,
			output:      true,
		},
		{
			description: "different keys don't match",
			a:           key1,
			b:           key2,
			output:      false,
		},
		{
			description: "empty keys match",
			a:           []byte{},
			b:           []byte{},
			output:      true,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			result := k.Compare(tt.a, tt.b)
			if got, want := result, tt.output; got != want {
				t.Errorf("result=%v, want=%v", got, want)
			}
		})
	}
}
