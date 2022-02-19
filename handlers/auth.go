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

func (s Server) requireAuthentication(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			s.authenticator.ClearSession(w)
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}
