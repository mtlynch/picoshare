package handlers

import (
	"context"
	"net/http"
)

var contextKeyIsAuthenticated = &contextKey{"is-authenticated"}

func (s Server) authPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.authenticator.StartSession(w, r)
	}
}

func (s Server) authDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.authenticator.ClearSession(w)
	}
}

func (s Server) checkAuthentication(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), contextKeyIsAuthenticated, s.authenticator.Authenticate((r)))
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s Server) requireAuthentication(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated((r.Context())) {
			s.authenticator.ClearSession(w)
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func isAuthenticated(ctx context.Context) bool {
	val, ok := ctx.Value(contextKeyIsAuthenticated).(bool)
	if !ok {
		return false
	}
	return val
}
