//go:build dev

package handlers

import "net/http"

func upgradeToHttps(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In dev-mode, this is a no-op, as we don't want to upgrade to HTTPS.
		h.ServeHTTP(w, r)
	})
}
