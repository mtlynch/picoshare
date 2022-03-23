package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/store"
)

func (s Server) entryGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		clientIp, err := clientIPFromRemoteAddr(r.RemoteAddr)
		if err != nil {
			log.Printf("failed to parse remote addr: %v -> %v", r.RemoteAddr, err)
			http.Error(w, "unrecognized source IP format", http.StatusBadRequest)
			return
		}

		entry, err := s.store.GetEntry(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		if !clientIp.Equal(entry.UploaderIP) {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "On demo instance, you can only download from the same IP as you uploaded", http.StatusForbidden)
			return
		}

		if entry.Filename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, entry.Filename))
		}

		w.Header().Set("Content-Type", string(entry.ContentType))

		http.ServeContent(w, r, string(entry.Filename), entry.Uploaded, entry.Reader)
	}
}
