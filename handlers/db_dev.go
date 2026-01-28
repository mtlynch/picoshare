//go:build dev

package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/mtlynch/picoshare/random"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

// addDevRoutes adds debug routes that we only use during development or e2e
// tests.
func (s *Server) addDevRoutes() {
	s.router.Use(assignSessionDB)
	s.router.HandleFunc("/api/debug/db/cleanup", s.cleanupPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/debug/db/per-session", dbPerSessionPost()).Methods(http.MethodPost)
}

const dbTokenCookieName = "db-token"

type (
	dbToken string

	dbSettings struct {
		isolateBySession bool
		lock             sync.RWMutex
	}
)

func (dbs *dbSettings) IsolateBySession() bool {
	dbs.lock.RLock()
	isolate := dbs.isolateBySession
	dbs.lock.RUnlock()
	return isolate
}

func (dbs *dbSettings) SetIsolateBySession(isolate bool) {
	dbs.lock.Lock()
	dbs.isolateBySession = isolate
	dbs.lock.Unlock()
	log.Printf("per-session database = %v", isolate)
}

var (
	sharedDBSettings dbSettings
	tokenToDB        map[dbToken]Store = map[dbToken]Store{}
	tokenToDBMutex   sync.RWMutex
)

func (s Server) getDB(r *http.Request) Store {
	if !sharedDBSettings.IsolateBySession() {
		return s.store
	}
	c, err := r.Cookie(dbTokenCookieName)
	if err != nil {
		panic(err)
	}

	tokenToDBMutex.RLock()
	defer tokenToDBMutex.RUnlock()
	return tokenToDB[dbToken(c.Value)]
}

func dbPerSessionPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sharedDBSettings.SetIsolateBySession(true)
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

// assignSessionDB provisions a session-specific database if per-session
// databases are enabled. If per-session databases are not enabled (the default)
// this is a no-op.
func assignSessionDB(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sharedDBSettings.IsolateBySession() {
			if _, err := r.Cookie(dbTokenCookieName); err != nil {
				token := dbToken(random.String(30, []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")))
				log.Printf("provisioning a new private database with token %s", token)
				createDBCookie(token, w)
				testDb := test_sqlite.New()
				tokenToDBMutex.Lock()
				tokenToDB[token] = &testDb
				tokenToDBMutex.Unlock()
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
