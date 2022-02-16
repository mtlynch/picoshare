package handlers

import "net/http"

func (s *Server) routes() {
	s.router.HandleFunc("/api/auth", s.authPost()).Methods(http.MethodPost)

	views := s.router.PathPrefix("/").Subrouter()
	views.HandleFunc("/", indexGet()).Methods(http.MethodGet)
	views.HandleFunc("/login", authGet()).Methods(http.MethodGet)

	static := s.router.PathPrefix("/").Subrouter()
	static.PathPrefix("/css/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/js/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
	static.PathPrefix("/third-party/").HandlerFunc(serveStaticResource()).Methods(http.MethodGet)
}
