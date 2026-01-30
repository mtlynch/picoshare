package handlers

import (
	"fmt"
	"log"
	"net/http"
)

// clearPost completely clears the database (for testing only).
// This endpoint is authentication-protected to prevent abuse.
func (s *Server) clearPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.ClearAll(); err != nil {
			log.Printf("database clear failed: %v", err)
			http.Error(w, fmt.Sprintf("database clear failed: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("database cleared successfully")
	}
}
