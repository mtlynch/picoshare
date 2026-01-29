package handlers

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

// entryPathHandler handles requests for entry paths like /-{id} and /!{id}.
// Go's ServeMux doesn't support wildcards with prefix characters, so we use a
// catch-all pattern and parse the ID from the path manually.
func (s Server) entryPathHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")

		// Check if path starts with - or ! (entry prefixes)
		if len(path) == 0 || (path[0] != '-' && path[0] != '!') {
			http.NotFound(w, r)
			return
		}

		// Extract ID (everything after the prefix until / or end)
		rest := path[1:] // Remove prefix character
		idStr := rest
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			idStr = rest[:slashIdx]
		}

		s.serveEntry(idStr, w, r)
	}
}

func (s Server) serveEntry(idStr string, w http.ResponseWriter, r *http.Request) {
	id, err := parseEntryID(idStr)
	if err != nil {
		log.Printf("error parsing ID: %v", err)
		http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
		return
	}

	entry, err := s.getDB(r).GetEntryMetadata(id)
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
	if contentType == "" || contentType == "application/octet-stream" {
		if inferred, err := inferContentTypeFromFilename(entry.Filename); err == nil {
			contentType = inferred
		}
	}
	w.Header().Set("Content-Type", contentType.String())

	entryFile, err := s.getDB(r).ReadEntryFile(id)
	if err != nil {
		log.Printf("error retrieving entry data with id %v: %v", id, err)
		http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, entry.Filename.String(), entry.Uploaded, entryFile)

	if err := recordDownload(s.getDB(r), entry.ID, s.clock.Now(), r.RemoteAddr, r.Header.Get("User-Agent")); err != nil {
		log.Printf("failed to record download of file %s: %v", id.String(), err)
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
