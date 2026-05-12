//go:build !dev

package handlers

import (
	"net/http"
)

func (s *Server) addDevRoutes() {
	// no-op
}

func (s Server) getDB(*http.Request) Store {
	return s.store
}
