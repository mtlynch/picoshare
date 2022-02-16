package handlers

import (
	"net/http"
)

func (s Server) authPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.authenticator.StartSession(w, r)
	}
}

func (s Server) isAuthenticated(r *http.Request) bool {
	return s.authenticator.Authenticate(r)
}
