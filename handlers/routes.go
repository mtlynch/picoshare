package handlers

import "net/http"

func (s *Server) routes() {
	views := s.router.PathPrefix("/").Subrouter()
	views.HandleFunc("/", s.indexGet()).Methods(http.MethodGet)
}
