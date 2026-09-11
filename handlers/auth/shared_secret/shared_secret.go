package shared_secret

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mtlynch/picoshare/handlers/auth/shared_secret/kdf"
	"github.com/mtlynch/picoshare/picoshare"
)

const (
	authCookieName = "sharedSecret"

	// maxSessionStartRequestBytes bounds the body of a request to start a
	// session. Even a passphrase of MaxPassphraseCodePoints code points that
	// JSON encodes entirely as escaped surrogate pairs fits well within it.
	maxSessionStartRequestBytes = 4096
)

var (
	// ErrInvalidCredentials indicates that the provided credentials are incorrect.
	ErrInvalidCredentials = errors.New("incorrect shared secret")

	// ErrMalformedRequest indicates that the request body is malformed.
	ErrMalformedRequest = errors.New("malformed request")
)

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	serverKey kdf.DerivedKey
}

type sessionStartRequest struct {
	Passphrase picoshare.Passphrase
}

// New creates a new SharedSecretAuthenticator.
func New(passphrase picoshare.Passphrase) SharedSecretAuthenticator {
	return SharedSecretAuthenticator{
		serverKey: kdf.DeriveKey(passphrase),
	}
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSessionStartRequestBytes)
	req, err := parseSessionStartRequest(r)
	if err != nil {
		if err == ErrMalformedRequest {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		}
		return
	}

	if serverKey, userKey := ssa.serverKey, kdf.DeriveKey(req.Passphrase); !serverKey.Equal(userKey) {
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

	serverKey := ssa.serverKey
	return serverKey.Equal(cookieKey)
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
		return sessionStartRequest{}, err
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
