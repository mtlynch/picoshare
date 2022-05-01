package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mtlynch/picoshare/v2/random"
)

func (s *Server) enforceContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cycle nonce.
		s.cspNonce = random.String(16, []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"))

		type cspDirective struct {
			name   string
			values []string
		}
		directives := []cspDirective{
			{
				name: "default-src",
				values: []string{
					"self",
				},
			},
			{
				name: "script-src",
				values: []string{
					"self",
					"nonce-" + s.cspNonce,
				},
			},
			{
				name: "style-src",
				values: []string{
					"self",
					"nonce-" + s.cspNonce,
				},
			},
		}
		policyParts := []string{}
		for _, directive := range directives {
			valuesFormatted := make([]string, len(directive.values))
			for i, value := range directive.values {
				valuesFormatted[i] = fmt.Sprintf("'%s'", value)
			}
			policyParts = append(policyParts, fmt.Sprintf("%s %s", directive.name, strings.Join(valuesFormatted, " ")))
		}
		policy := strings.Join(policyParts, "; ") + ";"

		w.Header().Set("Content-Security-Policy", policy)
		next.ServeHTTP(w, r)
	})
}
