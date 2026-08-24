package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

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
		router             *mux.Router
		authenticator      Authenticator
		store              Store
		spaceChecker       SpaceChecker
		collector          *garbagecollect.Collector
		clock              Clock
		stripImageMetadata bool
	}

	// Option customizes optional server behavior.
	Option func(*Server)
)

// WithImageMetadataStripping controls whether the server removes metadata (such
// as EXIF tags) from uploaded images before storing them. It is disabled by
// default.
func WithImageMetadataStripping(enabled bool) Option {
	return func(s *Server) {
		s.stripImageMetadata = enabled
	}
}

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator Authenticator, store Store, spaceChecker SpaceChecker, collector *garbagecollect.Collector, clock Clock, opts ...Option) Server {
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
		store:         store,
		spaceChecker:  spaceChecker,
		collector:     collector,
		clock:         clock,
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes()
	return s
}
