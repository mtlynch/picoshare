package handlers

import (
	"github.com/gorilla/mux"
)

type Server struct {
	router        *mux.Router
}

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New() Server {
	s := Server{
		router:        mux.NewRouter(),
	}

	s.routes()
	return s
}
