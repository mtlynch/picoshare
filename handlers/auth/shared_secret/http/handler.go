package http

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

// Authenticator handles HTTP authentication using shared secrets.
type Authenticator struct {
	kdf    kdf.KDF
	secret []byte
}

// New creates a new HTTP authenticator.
func New(sharedSecretKey string) (*Authenticator, error) {
	k := kdf.New()
	secret, err := k.DeriveFromKey([]byte(sharedSecretKey))
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		kdf:    k,
		secret: secret,
	}, nil
}

// StartSession begins an authenticated session.
func (a *Authenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	secret, err := a.sharedSecretFromRequest(r)
	if err != nil {
		switch err {
		case ErrMalformedRequest, ErrEmptyCredentials:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	if !a.kdf.Compare(secret, a.secret) {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	a.createCookie(w)
}

// Authenticate verifies if the request has valid authentication.
func (a *Authenticator) Authenticate(r *http.Request) bool {
	authCookie, err := r.Cookie(authCookieName)
	if err != nil {
		return false
	}

	secret, err := a.kdf.FromBase64(authCookie.Value)
	if err != nil {
		return false
	}

	return a.kdf.Compare(secret, a.secret)
}

// ClearSession removes the authentication cookie.
func (a *Authenticator) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func (a *Authenticator) sharedSecretFromRequest(r *http.Request) ([]byte, error) {
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

	return a.kdf.DeriveFromKey([]byte(body.SharedSecretKey))
}

func (a *Authenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    base64.StdEncoding.EncodeToString(a.secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
	})
}
