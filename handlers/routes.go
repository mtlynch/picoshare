package handlers

import "net/http"

func (s *Server) routes() {
	s.router.HandleFunc("/api/auth", s.authPost()).Methods(http.MethodPost)

	authenticatedApis := s.router.PathPrefix("/api").Subrouter()
	authenticatedApis.Use(s.requireAuthentication)
	authenticatedApis.HandleFunc("/entry", s.entryPost()).Methods(http.MethodPost)

	static := s.router.PathPrefix("/").Subrouter()
	static.PathPrefix("/css/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/js/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/third-party/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)

	views := s.router.PathPrefix("/").Subrouter()
	views.HandleFunc("/login", s.authGet()).Methods(http.MethodGet)
	views.PathPrefix("/!{id}").HandlerFunc(s.entryGet()).Methods(http.MethodGet)
	views.HandleFunc("/", s.indexGet()).Methods(http.MethodGet)
}
