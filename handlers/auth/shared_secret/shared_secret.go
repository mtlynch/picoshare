package shared_secret

import (
	"encoding/base64"
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
	CompareWithInput(inputKey []byte) bool
	CompareWithDerived(derivedKey []byte) bool
	GetDerivedKey() []byte
	FromBase64(b64encoded string) ([]byte, error)
	Compare(a, b []byte) bool
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
	inputKey, err := ssa.inputKeyFromRequest(r)
	if err != nil {
		switch err {
		case ErrMalformedRequest, ErrEmptyCredentials:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	if !ssa.kdf.CompareWithInput(inputKey) {
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

	derivedKey, err := ssa.kdf.FromBase64(authCookie.Value)
	if err != nil {
		return false
	}

	return ssa.kdf.CompareWithDerived(derivedKey)
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

func (ssa SharedSecretAuthenticator) inputKeyFromRequest(r *http.Request) ([]byte, error) {
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return nil, ErrMalformedRequest
	}

	if body.SharedSecretKey == "" {
		return nil, ErrEmptyCredentials
	}

	return []byte(body.SharedSecretKey), nil
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	derivedKey := ssa.kdf.GetDerivedKey()
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    base64.StdEncoding.EncodeToString(derivedKey),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
	})
}
