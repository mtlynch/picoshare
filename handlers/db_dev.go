//go:build dev

package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/test_sqlite"
)

// addDevRoutes adds debug routes that we only use during development or e2e
// tests.
func (s *Server) addDevRoutes() {
	s.router.Use(loadPerSessionDB)
	s.router.HandleFunc("/api/debug/db/cleanup", s.cleanupPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/debug/db/per-session", dbInPerSessionPost()).Methods(http.MethodPost)
}

const dbTokenCookieName = "db-token"

type dbToken string

var (
	// usePerSessionDB is a global flag that indicates whether to use a
	// per-session datastore. This is mainly useful for end-to-end tests.
	usePerSessionDB     bool
	usePerSessionDBLock sync.RWMutex
	tokenToDB           map[dbToken]store.Store = map[dbToken]store.Store{}
)

func (s Server) getDB(r *http.Request) store.Store {
	usePerSessionDBLock.RLock()
	defer usePerSessionDBLock.Unlock()
	if !usePerSessionDB {
		return s.store
	}
	c, err := r.Cookie(dbTokenCookieName)
	if err != nil {
		panic(err)
	}
	return tokenToDB[dbToken(c.Value)]
}

func dbInPerSessionPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usePerSessionDBLock.Lock()
		defer usePerSessionDBLock.Unlock()
		log.Printf("per-session database is now enabled")
		usePerSessionDB = true
	}
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

func loadPerSessionDB(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usePerSessionDBLock.RLock()
		defer usePerSessionDBLock.Unlock()
		if usePerSessionDB {
			if _, err := r.Cookie(dbTokenCookieName); err != nil {
				token := dbToken(random.String(30, []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")))
				log.Printf("provisioning a new private database with token %s", token)
				createDBCookie(token, w)
				tokenToDB[token] = test_sqlite.New()
			}
		}
		h.ServeHTTP(w, r)
	})
}

func createDBCookie(token dbToken, w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:  dbTokenCookieName,
		Value: string(token),
		Path:  "/",
	})
}
