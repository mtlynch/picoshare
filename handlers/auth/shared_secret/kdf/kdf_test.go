package kdf_test

import (
	"encoding/base64"
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
			err:         nil,
		},
		{
			description: "reject empty key",
			input:       []byte{},
			err:         kdf.ErrInvalidKey,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			k := kdf.New()
			key, err := k.DeriveFromKey(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if err == nil && len(key) == 0 {
				t.Errorf("key is empty")
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
		err         error
	}{
		{
			description: "accept valid base64",
			input:       validBase64,
			err:         nil,
		},
		{
			description: "reject empty string",
			input:       "",
			err:         kdf.ErrInvalidBase64,
		},
		{
			description: "reject invalid base64",
			input:       "not-base64!",
			err:         kdf.ErrInvalidBase64,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			key, err := k.FromBase64(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if err == nil && len(key) == 0 {
				t.Errorf("key is empty")
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
		want        bool
	}{
		{
			description: "same keys match",
			a:           key1,
			b:           key1Dup,
			want:        true,
		},
		{
			description: "different keys don't match",
			a:           key1,
			b:           key2,
			want:        false,
		},
		{
			description: "empty keys match",
			a:           []byte{},
			b:           []byte{},
			want:        true,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			if got, want := k.Compare(tt.a, tt.b), tt.want; got != want {
				t.Errorf("got=%v, want=%v", got, want)
			}
		})
	}
}
