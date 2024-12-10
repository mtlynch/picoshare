package handlers

import (
	"time"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/handlers/auth"
	"github.com/mtlynch/picoshare/v2/space"
)

type (
	SpaceChecker interface {
		Check() (space.Usage, error)
	}

	Clock interface {
		Now() time.Time
	}

	Server struct {
		router        *mux.Router
		authenticator auth.Authenticator
		store         Store
		spaceChecker  SpaceChecker
		collector     *garbagecollect.Collector
		clock         Clock
	}
)

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator auth.Authenticator, store Store, spaceChecker SpaceChecker, collector *garbagecollect.Collector, clock Clock) Server {
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
		store:         store,
		spaceChecker:  spaceChecker,
		collector:     collector,
		clock:         clock,
	}

	s.routes()
	return s
}
