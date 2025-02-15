package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/kdf"
)

const authCookieName = "sharedSecret"

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
		http.Error(w, "Invalid shared secret", http.StatusBadRequest)
		return
	}

	if !a.kdf.Compare(secret, a.secret) {
		http.Error(w, "Incorrect shared secret", http.StatusUnauthorized)
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
		Expires:  time.Unix(0, 0),
	})
}

func (a *Authenticator) sharedSecretFromRequest(r *http.Request) ([]byte, error) {
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		return nil, err
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
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})
}
