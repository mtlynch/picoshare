package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/kdf"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

func (s Server) entryGet() http.HandlerFunc {
	// Template for passphrase prompt page (consistent with site layout/styles).
	tPass := parseTemplates("templates/pages/file-protected.html")
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
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

		// If entry requires passphrase, render prompt page.
	if !entry.PassphraseKey.IsZero() {
			renderPassphrasePrompt(tPass, w, r, id, entry)
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
}

// entryAccessPost handles POST submissions to access a protected entry via passphrase.
func (s Server) entryAccessPost() http.HandlerFunc {
	tPass := parseTemplates("templates/pages/file-protected.html")
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
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

		// Parse form and validate passphrase when required.
		if !entry.PassphraseKey.IsZero() {
			if err := r.ParseForm(); err != nil {
				renderPassphrasePromptWithError(tPass, w, r, id, entry, "Invalid form submission")
				return
			}
			provided := r.PostFormValue("passphrase")
			if provided == "" {
				renderPassphrasePromptWithError(tPass, w, r, id, entry, "Passphrase required")
				return
			}
			derived, err := kdf.DeriveKeyFromSecret(provided)
			if err != nil {
				renderPassphrasePromptWithError(tPass, w, r, id, entry, "Invalid passphrase")
				return
			}
			if !entry.PassphraseKey.Equal(derived) {
				renderPassphrasePromptWithError(tPass, w, r, id, entry, "Incorrect passphrase")
				return
			}
		}

		// Serve the content upon successful validation.
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
}

func renderPassphrasePrompt(tPass *template.Template, w http.ResponseWriter, r *http.Request, id picoshare.EntryID, meta picoshare.UploadMetadata) {
	renderPassphrasePromptWithError(tPass, w, r, id, meta, "")
}

func renderPassphrasePromptWithError(tPass *template.Template, w http.ResponseWriter, r *http.Request, id picoshare.EntryID, meta picoshare.UploadMetadata, errorMsg string) {
	// Use the same site layout/styles via templates.
	_ = tPass.Execute(w, struct {
		commonProps
		PostURL  string
		Filename picoshare.Filename
		Error    string
	}{
		commonProps: makeCommonProps("PicoShare - Protected", r.Context()),
		PostURL:     r.URL.Path,
		Filename:    meta.Filename,
		Error:       errorMsg,
	})
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
