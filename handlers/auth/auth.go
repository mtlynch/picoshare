package auth

import (
	"net/http"
)

type Authenticator interface {
	StartSession(w http.ResponseWriter, r *http.Request)
	ClearSession(w http.ResponseWriter)
	Authenticate(r *http.Request) bool
}
