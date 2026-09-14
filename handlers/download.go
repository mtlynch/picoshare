package handlers

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

func (s Server) entryGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		entry, err := s.store.GetEntryMetadata(id)
		if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		// Serve response in a sandbox so that if a user uploads JavaScript, it
		// doesn't run in the same domain as the server.
		w.Header().Set("Content-Security-Policy", "sandbox")

		if entry.Filename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, entry.Filename))
		}

		contentType := entry.ContentType
		if contentType == "" || contentType == "application/octet-stream" {
			if inferred, err := inferContentTypeFromFilename(entry.Filename); err == nil {
				contentType = inferred
			}
		}
		w.Header().Set("Content-Type", contentType.String())

		entryFile, err := s.store.ReadEntryFile(id)
		if err != nil {
			log.Printf("error retrieving entry data with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		http.ServeContent(w, r, entry.Filename.String(), entry.Uploaded, entryFile)

		if err := recordDownload(s.store, entry.ID, s.now(), r.RemoteAddr, r.Header.Get("User-Agent")); err != nil {
			log.Printf("failed to record download of file %s: %v", id.String(), err)
		}
	}
}

func (s Server) entryUnlock() http.HandlerFunc {
	t := parseTemplates("templates/pages/download-passphrase.html")
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		if _, err := s.store.GetEntryMetadata(id); err != nil {
			if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
				http.Error(w, "entry not found", http.StatusNotFound)
				return
			}
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			IncorrectPassphrase bool
		}{
			commonProps:         makeCommonProps("PicoShare - Download", r.Context()),
			IncorrectPassphrase: false,
		})
	}
}

func inferContentTypeFromFilename(f picoshare.Filename) (picoshare.ContentType, error) {
	// For files that modern browser can play natively, infer the content type if
	// none was specified at upload time.
	if mimetype := mime.TypeByExtension(filepath.Ext(f.String())); mimetype != "" {
		return picoshare.ContentType(mimetype), nil
	}
	return picoshare.ContentType(""), errors.New("could not infer content type from filename")
}

func recordDownload(db Store, id picoshare.EntryID, t time.Time, remoteAddr, userAgent string) error {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

	return db.InsertEntryDownload(id, picoshare.DownloadRecord{
		Time:      t,
		ClientIP:  ip,
		UserAgent: userAgent,
	})
}
