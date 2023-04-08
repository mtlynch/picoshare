//go:build dev

package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

// addDevRoutes adds debug routes that we only use during development or e2e
// tests.
func (s *Server) addDevRoutes() {
	s.router.HandleFunc("/api/debug/db/cleanup", s.cleanupPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/debug/db/wipe", s.wipeDB()).Methods(http.MethodGet)
}

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

// wipeDB wipes the database back to a freshly initialized state.
func (s Server) wipeDB() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sqlStore, ok := s.store.(*sqlite.DB)
		if !ok {
			log.Fatalf("store is not SQLite, can't wipe database")
		}
		sqlStore.Clear()
	}
}
