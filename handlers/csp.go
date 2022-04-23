package handlers

import "net/http"

func enforceContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self';")
		next.ServeHTTP(w, r)
	})
}
