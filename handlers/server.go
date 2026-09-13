package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/garbagecollect"
)

type (
	Authenticator interface {
		StartSession(w http.ResponseWriter, r *http.Request)
		ClearSession(w http.ResponseWriter)
		Authenticate(r *http.Request) bool
	}

	Server struct {
		router        *mux.Router
		authenticator Authenticator
		store         Store
		checkSpace    SpaceCheckFunc
		collector     *garbagecollect.Collector
		now           NowFunc
	}
)

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator Authenticator, store Store, checkSpace SpaceCheckFunc, collector *garbagecollect.Collector, now NowFunc) Server {
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
		store:         store,
		checkSpace:    checkSpace,
		collector:     collector,
		now:           now,
	}

	s.routes()
	return s
}
