package shared_secret

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const authCookieName = "sharedSecret"

type (
	sharedSecret string

	SharedSecretAuthenticator struct {
		sharedSecret sharedSecret
	}
)

func New(sharedSecret string) (SharedSecretAuthenticator, error) {
	ss, err := parseSharedSecret(sharedSecret)
	if err != nil {
		return SharedSecretAuthenticator{}, err
	}

	return SharedSecretAuthenticator{
		sharedSecret: ss,
	}, nil
}

func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	ss, err := sharedSecretFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid shared secret", http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(ss), []byte(ssa.sharedSecret)) == 0 {
		http.Error(w, "Incorrect shared secret", http.StatusUnauthorized)
		return
	}

	ssa.createCookie(w)
}

func sharedSecretFromRequest(r *http.Request) (sharedSecret, error) {
	ar := struct {
		SharedSecret string `json:"sharedSecret"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ar)
	if err != nil {
		return "", err
	}

	return parseSharedSecret(ar.SharedSecret)
}

func (ssa SharedSecretAuthenticator) Authenticate(r *http.Request) bool {
	authCookie, err := r.Cookie(authCookieName)
	if err != nil {
		return false
	}

	ss, err := parseSharedSecret(authCookie.Value)
	if err != nil {
		return false
	}

	if ss != ssa.sharedSecret {
		return false
	}

	return true
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    string(ssa.sharedSecret),
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})
}

func (ssa SharedSecretAuthenticator) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
}

func parseSharedSecret(s string) (sharedSecret, error) {
	if len(s) == 0 {
		return sharedSecret(""), errors.New("invalid shared secret")
	}
	return sharedSecret(s), nil
}
