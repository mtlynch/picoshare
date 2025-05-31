package kdf_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

func TestDeriveFromKey(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       []byte
		output      []byte
		err         error
	}{
		{
			description: "accept valid key",
			input:       []byte("mysecret"),
			output:      []byte{}, // We can't predict the exact output, but we'll verify it's not empty
			err:         nil,
		},
		{
			description: "reject empty key",
			input:       []byte{},
			output:      nil,
			err:         kdf.ErrInvalidKey,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, string(tt.input)), func(t *testing.T) {
			k := kdf.New()
			key, err := k.DeriveFromKey(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if err == nil {
				if len(key) == 0 {
					t.Errorf("key is empty, expected non-empty output")
				}
			} else if key != nil {
				t.Errorf("key=%v, want=nil when error occurs", key)
			}
		})
	}
}

func TestFromBase64(t *testing.T) {
	k := kdf.New()
	validKey, _ := k.DeriveFromKey([]byte("test"))
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
	k := kdf.New()
	key1, _ := k.DeriveFromKey([]byte("test1"))
	key2, _ := k.DeriveFromKey([]byte("test2"))
	key1Dup, _ := k.DeriveFromKey([]byte("test1"))

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
