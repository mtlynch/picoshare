package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

func enforceContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directives := map[string][]string{
			"default-src": {
				"self",
			},
			"script-src": {
				"self",
				// HTML custom elements require unsafe-inline.
				"unsafe-inline",
			},
			"style-src": {
				"self",
				// HTML custom elements require unsafe-inline.
				"unsafe-inline",
			},
		}
		policyParts := []string{}
		for directive, keywords := range directives {
			keywordsFormatted := make([]string, len(keywords))
			for i, keyword := range keywords {
				keywordsFormatted[i] = fmt.Sprintf("'%s'", keyword)
			}
			policyParts = append(policyParts, fmt.Sprintf("%s %s", directive, strings.Join(keywordsFormatted, " ")))
		}
		policy := strings.Join(policyParts, "; ")

		w.Header().Set("Content-Security-Policy", policy)
		next.ServeHTTP(w, r)
	})
}
