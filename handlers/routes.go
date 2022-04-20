package handlers

import "net/http"

func (s *Server) routes() {
	s.router.HandleFunc("/api/auth", s.authPost()).Methods(http.MethodPost)
	s.router.HandleFunc("/api/auth", s.authDelete()).Methods(http.MethodDelete)

	authenticatedApis := s.router.PathPrefix("/api").Subrouter()
	authenticatedApis.Use(s.requireAuthentication)
	authenticatedApis.HandleFunc("/entry", s.entryPost()).Methods(http.MethodPost)
	authenticatedApis.HandleFunc("/entry/{id}", s.entryDelete()).Methods(http.MethodDelete)

	static := s.router.PathPrefix("/").Subrouter()
	static.PathPrefix("/css/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/js/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/third-party/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)

	authenticatedViews := s.router.PathPrefix("/").Subrouter()
	authenticatedViews.Use(s.requireAuthentication)
	authenticatedViews.HandleFunc("/files", s.fileIndexGet()).Methods(http.MethodGet)

	views := s.router.PathPrefix("/").Subrouter()
	views.Use(upgradeToHttps)
	views.HandleFunc("/login", s.authGet()).Methods(http.MethodGet)
	views.PathPrefix("/!{id}").HandlerFunc(s.entryGet()).Methods(http.MethodGet)
	views.PathPrefix("/!{id}/{filename}").HandlerFunc(s.entryGet()).Methods(http.MethodGet)
	views.HandleFunc("/", s.indexGet()).Methods(http.MethodGet)
}
