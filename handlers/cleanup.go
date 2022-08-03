package handlers

import (
	"fmt"
	"log"
	"net/http"
)

// cleanupPost is mainly for debugging/testing, as the garbagecollect package
// performs this action on a regular schedule.
func (s *Server) cleanupPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.collector.Collect(); err != nil {
			log.Printf("garbage collection failed: %v", err)
			http.Error(w, fmt.Sprintf("garbage collection failed: %v", err), http.StatusInternalServerError)
			return
		}
	}
}
