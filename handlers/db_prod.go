//go:build !dev

package handlers

import (
	"net/http"

	"github.com/mtlynch/picoshare/v2/store"
)

func (s *Server) addDevRoutes() {
	// no-op
}

func (s Server) getDB(*http.Request) store.Store {
	return s.store
}
