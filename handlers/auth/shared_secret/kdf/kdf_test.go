package kdf_test

import (
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

func TestDeriveKeyFromSecret(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		err         error
	}{
		{
			description: "accept valid secret",
			input:       "mysecret",
			err:         nil,
		},
		{
			description: "reject empty secret",
			input:       "",
			err:         kdf.ErrInvalidSecret,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			_, err := kdf.DeriveKeyFromSecret(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
		})
	}
}

func TestKeyComparison(t *testing.T) {
	originalKey, err := kdf.DeriveKeyFromSecret("test")
	if got, want := err, error(nil); got != want {
		t.Fatalf("failed to derive key: err=%v, want=%v", got, want)
	}

	t.Run("same secret creates matching keys", func(t *testing.T) {
		sameKey, err := kdf.DeriveKeyFromSecret("test")
		if got, want := err, error(nil); got != want {
			t.Fatalf("failed to derive second key: err=%v, want=%v", got, want)
		}
		if got, want := originalKey.Equal(sameKey), true; got != want {
			t.Errorf("key comparison=%v, want=%v", got, want)
		}
	})

	t.Run("different secrets don't match", func(t *testing.T) {
		otherKey, err := kdf.DeriveKeyFromSecret("different-secret")
		if got, want := err, error(nil); got != want {
			t.Fatalf("failed to derive third key: err=%v, want=%v", got, want)
		}
		if got, want := originalKey.Equal(otherKey), false; got != want {
			t.Errorf("key comparison=%v, want=%v", got, want)
		}
	})

	t.Run("comparison with empty key panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("comparing an empty key should panic")
			}
		}()

		originalKey.Equal(kdf.DerivedKey{})
	})
}

func TestSerializeDeserialize(t *testing.T) {
	key, err := kdf.DeriveKeyFromSecret("test")
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	deserializedKey, err := kdf.DeserializeKey(key.Serialize())
	if err != nil {
		t.Errorf("failed to deserialize: %v", err)
	}

	if !key.Equal(deserializedKey) {
		t.Errorf("deserialized key doesn't match original")
	}
}

func TestSerializeEmptyKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("serializing an empty key should panic")
		}
	}()

	kdf.DerivedKey{}.Serialize()
}
