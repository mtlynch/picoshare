package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/picoshare"
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

		entry, err := s.getDB(r).GetEntry(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		if entry.Filename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, entry.Filename))
		}

		contentType := entry.ContentType
		if contentType == "" {
			if inferred, err := inferContentTypeFromFilename(entry.Filename); err == nil {
				contentType = inferred
			}
		}
		w.Header().Set("Content-Type", string(contentType))

		http.ServeContent(w, r, string(entry.Filename), entry.Uploaded, entry.Reader)
	}
}

func inferContentTypeFromFilename(f picoshare.Filename) (picoshare.ContentType, error) {
	// For files that modern browser can play natively, infer the content type if
	// none was specified at upload time.
	switch filepath.Ext(f.String()) {
	case ".mp4":
		return picoshare.ContentType("video/mp4"), nil
	case ".mp3":
		return picoshare.ContentType("audio/mpeg"), nil
	default:
		return picoshare.ContentType(""), errors.New("could not infer content type from filename")
	}
}
