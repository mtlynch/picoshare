package shared_secret

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const authCookieName = "sharedSecret"

type (
	sharedSecret []byte

	SharedSecretAuthenticator struct {
		sharedSecret sharedSecret
	}
)

func New(sharedSecretKey string) (SharedSecretAuthenticator, error) {
	ss, err := sharedSecretFromKey([]byte(sharedSecretKey))
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

	if !sharedSecretsEqual(ss, ssa.sharedSecret) {
		http.Error(w, "Incorrect shared secret", http.StatusUnauthorized)
		return
	}

	ssa.createCookie(w)
}

func sharedSecretFromRequest(r *http.Request) (sharedSecret, error) {
	body := struct {
		SharedSecretKey string `json:"sharedSecretKey"`
	}{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		return sharedSecret{}, err
	}

	return sharedSecretFromKey([]byte(body.SharedSecretKey))
}

func (ssa SharedSecretAuthenticator) Authenticate(r *http.Request) bool {
	authCookie, err := r.Cookie(authCookieName)
	if err != nil {
		return false
	}

	ss, err := sharedSecretFromBase64(authCookie.Value)
	if err != nil {
		return false
	}

	return sharedSecretsEqual(ss, ssa.sharedSecret)
}

func (ssa SharedSecretAuthenticator) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    base64.StdEncoding.EncodeToString(ssa.sharedSecret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})
}

func (ssa SharedSecretAuthenticator) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
}

func sharedSecretFromKey(key []byte) (sharedSecret, error) {
	if len(key) == 0 {
		return sharedSecret{}, errors.New("invalid shared secret")
	}

	// These would be insecure values for storing a database of user credentials,
	// but we're only storing a single password, so it's not important to have
	// random salt or high iteration rounds.
	staticSalt := []byte{1, 2, 3, 4}
	iter := 100

	dk := pbkdf2.Key(key, staticSalt, iter, 32, sha256.New)

	return sharedSecret(dk), nil
}

func sharedSecretFromBase64(b64encoded string) (sharedSecret, error) {
	if len(b64encoded) == 0 {
		return sharedSecret{}, errors.New("invalid shared secret")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64encoded)
	if err != nil {
		return sharedSecret{}, err
	}

	return sharedSecret(decoded), nil
}

func sharedSecretsEqual(a, b sharedSecret) bool {
	return subtle.ConstantTimeCompare(a, b) != 0
}
