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

// DownloadPassphraseHash is a per-file passphrase verifier.
type DownloadPassphraseHash struct {
	salt []byte
	key  []byte
}

func HashDownloadPassphrase(passphrase Passphrase) (DownloadPassphraseHash, error) {
	salt := make([]byte, downloadPassphraseSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return DownloadPassphraseHash{}, fmt.Errorf("failed to generate passphrase salt: %w", err)
	}
	return DownloadPassphraseHash{
		salt: salt,
		key:  deriveDownloadPassphraseKey(passphrase, salt),
	}, nil
}

func ParseDownloadPassphraseHash(encoded string) (DownloadPassphraseHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2-sha256" || parts[1] != "v=1" || parts[2] != "i="+strconv.Itoa(downloadPassphraseIterations) {
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
	return DownloadPassphraseHash{salt: salt, key: key}, nil
}

func (h DownloadPassphraseHash) Matches(passphrase Passphrase) bool {
	if len(h.key) == 0 {
		panic("cannot match against an uninitialized download passphrase hash")
	}
	return subtle.ConstantTimeCompare(h.key, deriveDownloadPassphraseKey(passphrase, h.salt)) == 1
}

func (h DownloadPassphraseHash) Encoded() string {
	if len(h.key) == 0 {
		panic("cannot encode an uninitialized download passphrase hash")
	}
	return strings.Join([]string{
		"pbkdf2-sha256",
		"v=1",
		"i=" + strconv.Itoa(downloadPassphraseIterations),
		base64.RawStdEncoding.EncodeToString(h.salt),
		base64.RawStdEncoding.EncodeToString(h.key),
	}, "$")
}

func deriveDownloadPassphraseKey(passphrase Passphrase, salt []byte) []byte {
	return pbkdf2.Key(passphrase.Bytes(), salt, downloadPassphraseIterations, downloadPassphraseKeyLength, sha256.New)
}
