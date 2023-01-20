package unsecured

import (
	"net/http"
)

type UnsecuredAuthenticator struct{}

func New() UnsecuredAuthenticator {
	return UnsecuredAuthenticator{}
}

func (ssa UnsecuredAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {}

func (ssa UnsecuredAuthenticator) Authenticate(r *http.Request) bool {
	return true
}

func (ssa UnsecuredAuthenticator) ClearSession(w http.ResponseWriter) {}
