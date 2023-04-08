//go:build dev

package handlers

import (
	"log"
	"net/http"

	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

// addDevRoutes adds debug routes that we only use during development or e2e
// tests.
func (s *Server) addDevRoutes() {
	s.router.HandleFunc("/api/debug/db/wipe", s.wipeDB()).Methods(http.MethodGet)
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
