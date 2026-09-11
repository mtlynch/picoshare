package shared_secret

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mtlynch/picoshare/handlers/auth/shared_secret/kdf"
	"github.com/mtlynch/picoshare/picoshare"
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

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	serverKey kdf.DerivedKey
}

// New creates a new SharedSecretAuthenticator.
func New(passphrase picoshare.Passphrase) SharedSecretAuthenticator {
	serverKey := kdf.DeriveKey(passphrase)
	return SharedSecretAuthenticator{
		serverKey: serverKey,
	}
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	passphrase, err := parseSessionStartRequest(r)
	if err != nil {
		switch err {
		case ErrMalformedRequest, ErrEmptyCredentials:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	if serverKey, userKey := ssa.serverKey, kdf.DeriveKey(passphrase.Passphrase); !serverKey.Equal(userKey) {
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

	cookieKey, err := kdf.DeserializeKey(authCookie.Value)
	if err != nil {
		return false
	}

	return ssa.serverKey.Equal(cookieKey)
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

type sessionStartRequest struct {
	Passphrase picoshare.Passphrase
}

func parseSessionStartRequest(r *http.Request) (sessionStartRequest, error) {
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return sessionStartRequest{}, ErrMalformedRequest
	}
	passphrase, err := picoshare.NewPassphrase(body.SharedSecretKey)
	if err != nil {
		return sessionStartRequest{}, ErrEmptyCredentials
	}
	return sessionStartRequest{Passphrase: passphrase}, nil
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    ssa.serverKey.Serialize(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
	})
}
