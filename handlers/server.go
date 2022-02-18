package handlers

import (
	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/v2/handlers/auth"
)

type Server struct {
	router        *mux.Router
	authenticator auth.Authenticator
}

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator auth.Authenticator) Server {
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
	}

	s.routes()
	return s
}
