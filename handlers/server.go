package handlers

import (
	"sync"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/handlers/auth"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

type (
	syncedSettings struct {
		settings picoshare.Settings
		mu       *sync.RWMutex
	}

	Server struct {
		router        *mux.Router
		authenticator auth.Authenticator
		store         store.Store
		collector     *garbagecollect.Collector
		settings      *syncedSettings
	}
)

// Router returns the underlying router interface for the server.
func (s Server) Router() *mux.Router {
	return s.router
}

// New creates a new server with all the state it needs to satisfy HTTP
// requests.
func New(authenticator auth.Authenticator, store store.Store, collector *garbagecollect.Collector) (Server, error) {
	settings, err := store.ReadSettings()
	if err != nil {
		return Server{}, err
	}
	s := Server{
		router:        mux.NewRouter(),
		authenticator: authenticator,
		store:         store,
		collector:     collector,
		settings: &syncedSettings{
			settings: settings,
			mu:       &sync.RWMutex{},
		},
	}

	s.routes()
	return s, nil
}

func (ss syncedSettings) Get() picoshare.Settings {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.settings
}

func (ss *syncedSettings) Update(s picoshare.Settings) {
	ss.mu.Lock()
	ss.settings = s
	ss.mu.Unlock()
}
