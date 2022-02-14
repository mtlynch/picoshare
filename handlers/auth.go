package handlers

import (
	"net/http"
)

func (s Server) authPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Actually authenticate
	}
}
