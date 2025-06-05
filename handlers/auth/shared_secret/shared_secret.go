package shared_secret

import (
	"net/http"

	httpAuth "github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret/http"
)

// SharedSecretAuthenticator handles authentication using a shared secret.
type SharedSecretAuthenticator struct {
	authenticator *httpAuth.Authenticator
}

// New creates a new SharedSecretAuthenticator.
func New(sharedSecretKey string) (SharedSecretAuthenticator, error) {
	authenticator, err := httpAuth.New(sharedSecretKey)
	if err != nil {
		return SharedSecretAuthenticator{}, err
	}

	return SharedSecretAuthenticator{
		authenticator: authenticator,
	}, nil
}

// StartSession begins an authenticated session.
func (ssa SharedSecretAuthenticator) StartSession(w http.ResponseWriter, r *http.Request) {
	ssa.authenticator.StartSession(w, r)
}

// Authenticate verifies if the request has valid authentication.
func (ssa SharedSecretAuthenticator) Authenticate(r *http.Request) bool {
	return ssa.authenticator.Authenticate(r)
}

// ClearSession removes the authentication cookie.
func (ssa SharedSecretAuthenticator) ClearSession(w http.ResponseWriter) {
	ssa.authenticator.ClearSession(w)
}
