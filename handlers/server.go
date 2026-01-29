package handlers

import (
	"net/http"
	"time"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/space"
)

type (
	SpaceChecker interface {
		Check() (space.Usage, error)
	}

	Clock interface {
		Now() time.Time
	}

	Authenticator interface {
		StartSession(w http.ResponseWriter, r *http.Request)
		ClearSession(w http.ResponseWriter)
		Authenticate(r *http.Request) bool
	}

	Server struct {
		router        *http.ServeMux
		authenticator Authenticator
		store         Store
		spaceChecker  SpaceChecker
		collector     *garbagecollect.Collector
		clock         Clock
	}
)

// Router returns the underlying router interface for the server.
func (s Server) Router() *http.ServeMux {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator Authenticator, store Store, spaceChecker SpaceChecker, collector *garbagecollect.Collector, clock Clock) Server {
	s := Server{
		router:        http.NewServeMux(),
		authenticator: authenticator,
		store:         store,
		spaceChecker:  spaceChecker,
		collector:     collector,
		clock:         clock,
	}

	s.routes()
	return s
}
