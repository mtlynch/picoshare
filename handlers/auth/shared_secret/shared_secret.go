package shared_secret

import (
	"net/http"

	httpAuth "github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/http"
)

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	auth *httpAuth.Authenticator
}

// New creates a new SharedSecretAuthenticator.
func New(sharedSecretKey string) (SharedSecretAuthenticator, error) {
	auth, err := httpAuth.New(sharedSecretKey)
	if err != nil {
		return SharedSecretAuthenticator{}, err
	}

	return SharedSecretAuthenticator{
		auth: auth,
	}, nil
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	ssa.auth.StartSession(w, r)
}

// Authenticate verifies if the request has valid authentication.
func (ssa SharedSecretAuthenticator) Authenticate(r *http.Request) bool {
	return ssa.auth.Authenticate(r)
}

// ClearSession removes the authentication cookie.
func (ssa SharedSecretAuthenticator) ClearSession(w http.ResponseWriter) {
	ssa.auth.ClearSession(w)
}
