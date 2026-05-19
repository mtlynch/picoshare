package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/handlers/parse"
)

func (s Server) friendlyLinksDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parse.FriendlyName(mux.Vars(r)["friendlyName"])
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid friendly name: %v", err), http.StatusBadRequest)
			return
		}

		if err := s.getDB(r).DeleteFriendlyLink(name); err != nil {
			log.Printf("failed to delete friendly link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete friendly link: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) friendlyLinksEnableDisable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parse.FriendlyName(mux.Vars(r)["friendlyName"])
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid friendly name: %v", err), http.StatusBadRequest)
			return
		}

		fl, err := s.getDB(r).GetFriendlyLink(name)
		if err != nil {
			log.Printf("failed to get friendly link %s: %v", name, err)
			http.Error(w, fmt.Sprintf("Friendly link with name %s not found: %v", name, err), http.StatusNotFound)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/enable") {
			fl.IsDisabled = false
		} else {
			fl.IsDisabled = true
		}

		if err := s.getDB(r).UpdateFriendlyLink(fl); err != nil {
			log.Printf("failed to change friendly link enabled state: %v", err)
			http.Error(w, fmt.Sprintf("Failed to change friendly link enabled state: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
