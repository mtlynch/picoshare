package handlers

import (
	"net/http"
)

func (s Server) indexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from picoshare"))
	}
}
