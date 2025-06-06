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
	DeriveFromKey(key []byte) ([]byte, error)
	FromBase64(b64encoded string) ([]byte, error)
	Compare(a, b []byte) bool
}

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	kdf    KDF
	secret []byte
}

// New creates a new SharedSecretAuthenticator.
func New(sharedSecretKey string) (SharedSecretAuthenticator, error) {
	k := kdf.New()
	secret, err := k.DeriveFromKey([]byte(sharedSecretKey))
	if err != nil {
		return SharedSecretAuthenticator{}, err
	}

	return SharedSecretAuthenticator{
		kdf:    k,
		secret: secret,
	}, nil
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	secret, err := ssa.sharedSecretFromRequest(r)
	if err != nil {
		switch err {
		case ErrMalformedRequest, ErrEmptyCredentials:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	if !ssa.kdf.Compare(secret, ssa.secret) {
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

	secret, err := ssa.kdf.FromBase64(authCookie.Value)
	if err != nil {
		return false
	}

	return ssa.kdf.Compare(secret, ssa.secret)
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

func (ssa SharedSecretAuthenticator) sharedSecretFromRequest(r *http.Request) ([]byte, error) {
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

	return ssa.kdf.DeriveFromKey([]byte(body.SharedSecretKey))
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    base64.StdEncoding.EncodeToString(ssa.secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
	})
}
