package kdf_test

import (
	"testing"

	"github.com/mtlynch/picoshare/handlers/auth/shared_secret/kdf"
	"github.com/mtlynch/picoshare/picoshare"
)

func TestKeyComparison(t *testing.T) {
	originalKey := kdf.DeriveKey(mustCreatePassphrase(t, "test"))

	t.Run("same secret creates matching keys", func(t *testing.T) {
		sameKey := kdf.DeriveKey(mustCreatePassphrase(t, "test"))
		if got, want := originalKey.Equal(sameKey), true; got != want {
			t.Errorf("key comparison=%v, want=%v", got, want)
		}
	})

	t.Run("different secrets don't match", func(t *testing.T) {
		otherKey := kdf.DeriveKey(mustCreatePassphrase(t, "different-secret"))
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
	key := kdf.DeriveKey(mustCreatePassphrase(t, "test"))

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

func mustCreatePassphrase(t *testing.T, raw string) picoshare.Passphrase {
	t.Helper()
	passphrase, err := picoshare.NewPassphrase(raw)
	if err != nil {
		t.Fatalf("failed to create passphrase: %v", err)
	}
	return passphrase
}
