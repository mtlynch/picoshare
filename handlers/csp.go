package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/mtlynch/picoshare/v2/random"
)

type contextKey struct {
	name string
}

var contextKeyCSPNonce = &contextKey{"csp-nonce"}

func enforceContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := base64.StdEncoding.EncodeToString(random.Bytes(16))

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
					"nonce-" + nonce,
				},
			},
			{
				name: "style-src",
				values: []string{
					"self",
					"nonce-" + nonce,
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

		ctx := context.WithValue(r.Context(), contextKeyCSPNonce, nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func cspNonce(ctx context.Context) string {
	key, ok := ctx.Value(contextKeyCSPNonce).(string)
	if !ok {
		panic("CSP nonce is missing from request context")
	}
	return key
}
