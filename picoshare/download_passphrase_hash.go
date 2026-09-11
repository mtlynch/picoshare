package picoshare

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	downloadPassphraseIterations = 600000
	downloadPassphraseSaltLength = 16
	downloadPassphraseKeyLength  = 32
)

var ErrInvalidDownloadPassphraseHash = errors.New("invalid download passphrase hash")

// DownloadPassphraseHash is an opaque, serialized per-file passphrase verifier.
type DownloadPassphraseHash struct {
	encoded string
}

func HashDownloadPassphrase(passphrase Passphrase) (DownloadPassphraseHash, error) {
	salt := make([]byte, downloadPassphraseSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return DownloadPassphraseHash{}, fmt.Errorf("failed to generate passphrase salt: %w", err)
	}
	key := pbkdf2.Key(passphrase.Bytes(), salt, downloadPassphraseIterations, downloadPassphraseKeyLength, sha256.New)
	return newDownloadPassphraseHash(salt, key), nil
}

func ParseDownloadPassphraseHash(encoded string) (DownloadPassphraseHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2-sha256" || parts[1] != "v=1" || parts[2] != "i=600000" {
		return DownloadPassphraseHash{}, ErrInvalidDownloadPassphraseHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) != downloadPassphraseSaltLength {
		return DownloadPassphraseHash{}, ErrInvalidDownloadPassphraseHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(key) != downloadPassphraseKeyLength {
		return DownloadPassphraseHash{}, ErrInvalidDownloadPassphraseHash
	}
	return DownloadPassphraseHash{encoded: encoded}, nil
}

func (h DownloadPassphraseHash) Matches(passphrase Passphrase) bool {
	parsed, err := ParseDownloadPassphraseHash(h.encoded)
	if err != nil {
		return false
	}
	parts := strings.Split(parsed.encoded, "$")
	salt, _ := base64.RawStdEncoding.DecodeString(parts[3])
	expected, _ := base64.RawStdEncoding.DecodeString(parts[4])
	actual := pbkdf2.Key(passphrase.Bytes(), salt, downloadPassphraseIterations, downloadPassphraseKeyLength, sha256.New)
	return subtle.ConstantTimeCompare(expected, actual) == 1
}

func (h DownloadPassphraseHash) Encoded() string { return h.encoded }

func newDownloadPassphraseHash(salt, key []byte) DownloadPassphraseHash {
	return DownloadPassphraseHash{encoded: strings.Join([]string{
		"pbkdf2-sha256", "v=1", "i=" + strconv.Itoa(downloadPassphraseIterations),
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key),
	}, "$")}
}
