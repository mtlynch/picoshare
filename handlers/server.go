package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/space"
)

type (
	SpaceChecker interface {
		Check() (space.Usage, error)
	}

	Authenticator interface {
		StartSession(w http.ResponseWriter, r *http.Request)
		ClearSession(w http.ResponseWriter)
		Authenticate(r *http.Request) bool
	}

	Server struct {
		router        *mux.Router
		authenticator Authenticator
		store         Store
		spaceChecker  SpaceChecker
		collector     *garbagecollect.Collector
		now           picoshare.NowFunc
	}
)

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator Authenticator, store Store, spaceChecker SpaceChecker, collector *garbagecollect.Collector, now picoshare.NowFunc) Server {
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
		store:         store,
		spaceChecker:  spaceChecker,
		collector:     collector,
		now:           now,
	}

	s.routes()
	return s
}
