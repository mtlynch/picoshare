package shared_secret

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

const authCookieName = "sharedSecret"

var (
	// ErrInvalidCredentials indicates that the provided credentials are incorrect.
	ErrInvalidCredentials = errors.New("incorrect shared secret")

	// ErrEmptyCredentials indicates that no credentials were provided.
	ErrEmptyCredentials = errors.New("invalid shared secret")

	// ErrMalformedRequest indicates that the request body is malformed.
	ErrMalformedRequest = errors.New("malformed request")
)

// KDF defines the interface for key derivation operations.
type KDF interface {
	Compare(key kdf.DerivedKey) bool
	Serialize() string
	Deserialize(s string) (kdf.DerivedKey, error)
}

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	kdf KDF
}

// New creates a new SharedSecretAuthenticator.
func New(sharedSecretKey string) (SharedSecretAuthenticator, error) {
	k, err := kdf.New([]byte(sharedSecretKey))
	if err != nil {
		return SharedSecretAuthenticator{}, err
	}

	return SharedSecretAuthenticator{
		kdf: k,
	}, nil
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	inputKeyString, err := ssa.inputKeyFromRequest(r)
	if err != nil {
		switch err {
		case ErrMalformedRequest, ErrEmptyCredentials:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	// Convert raw input to RawKey, then derive to DerivedKey for comparison
	rawKey := kdf.NewRawKey(inputKeyString)
	derivedKey := rawKey.Derive(ssa.kdf.(*kdf.Pbkdf2KDF))

	if !ssa.kdf.Compare(derivedKey) {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	ssa.createCookie(w)
}

// Authenticate verifies if the request has valid authentication.
func (ssa SharedSecretAuthenticator) Authenticate(r *http.Request) bool {
	authCookie, err := r.Cookie(authCookieName)
	if err != nil {
		return false
	}

	derivedKey, err := ssa.kdf.Deserialize(authCookie.Value)
	if err != nil {
		return false
	}

	return ssa.kdf.Compare(derivedKey)
}

// ClearSession removes the authentication cookie.
func (ssa SharedSecretAuthenticator) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func (ssa SharedSecretAuthenticator) inputKeyFromRequest(r *http.Request) (string, error) {
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return "", ErrMalformedRequest
	}

	if body.SharedSecretKey == "" {
		return "", ErrEmptyCredentials
	}

	return body.SharedSecretKey, nil
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    ssa.kdf.Serialize(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
	})
}
