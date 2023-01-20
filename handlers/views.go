package handlers

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"path"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

type commonProps struct {
	Title           string
	IsAuthenticated bool
	CspNonce        string
}

func (s Server) indexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r.Context()) {
			s.uploadGet()(w, r)
			return
		}
		if err := renderTemplate(w, "index.html", struct {
			commonProps
		}{
			commonProps: makeCommonProps("PicoShare", r.Context()),
		}, template.FuncMap{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestLinkIndexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := s.store.GetGuestLinks()
		if err != nil {
			log.Printf("failed to retrieve guest links: %v", err)
			http.Error(w, "Failed to retrieve guest links", http.StatusInternalServerError)
			return
		}

		sort.Slice(links, func(i, j int) bool {
			return links[i].Created.After(links[j].Created)
		})

		if err := renderTemplate(w, "guest-link-index.html", struct {
			commonProps
			GuestLinks []picoshare.GuestLink
		}{
			commonProps: makeCommonProps("PicoShare - Guest Links", r.Context()),
			GuestLinks:  links,
		}, template.FuncMap{
			"formatDate": func(t time.Time) string {
				return t.Format("2006-01-02")
			},
			"formatSizeLimit": func(limit picoshare.GuestUploadMaxFileBytes) string {
				if limit == picoshare.GuestUploadUnlimitedFileSize {
					return "Unlimited"
				}
				b := uint64(*limit)
				const unit = 1024

				if b < unit {
					return fmt.Sprintf("%d B", b)
				}
				div, exp := int64(unit), 0
				for n := b / unit; n >= unit; n /= unit {
					div *= unit
					exp++
				}
				return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "kMGTPE"[exp])
			},
			"formatCountLimit": func(limit picoshare.GuestUploadCountLimit) string {
				if limit == picoshare.GuestUploadUnlimitedFileUploads {
					return "Unlimited"
				}
				return fmt.Sprintf("%d", int(*limit))
			},
			"formatExpiration": func(et picoshare.ExpirationTime) string {
				if et == picoshare.NeverExpire {
					return "Never"
				}
				t := time.Time(et)
				delta := time.Until(t)
				suffix := ""
				if delta.Seconds() < 0 {
					suffix = " ago"
				}
				return fmt.Sprintf("%s (%.0f days%s)", t.Format("2006-01-02"), math.Abs(delta.Hours())/24, suffix)
			},
			"isActive": func(gl picoshare.GuestLink) bool {
				return gl.IsActive()
			},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestLinksNewGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type expirationOption struct {
			FriendlyName string
			Expiration   time.Time
			IsDefault    bool
		}
		if err := renderTemplate(w, "guest-link-create.html", struct {
			commonProps
			ExpirationOptions []expirationOption
		}{
			commonProps: makeCommonProps("PicoShare - New Guest Link", r.Context()),
			ExpirationOptions: []expirationOption{
				{"1 day", time.Now().AddDate(0, 0, 1), false},
				{"7 days", time.Now().AddDate(0, 0, 7), false},
				{"30 days", time.Now().AddDate(0, 0, 30), false},
				{"1 year", time.Now().AddDate(1, 0, 0), false},
				{"Never", time.Time(picoshare.NeverExpire), true},
			},
		}, template.FuncMap{
			"formatExpiration": func(t time.Time) string {
				return t.Format(time.RFC3339)
			}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileIndexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		em, err := s.store.GetEntriesMetadata()
		if err != nil {
			log.Printf("failed to retrieve entries metadata: %v", err)
			http.Error(w, "failed to retrieve file index", http.StatusInternalServerError)
			return
		}
		sort.Slice(em, func(i, j int) bool {
			return em[i].Uploaded.After(em[j].Uploaded)
		})
		if err := renderTemplate(w, "file-index.html", struct {
			commonProps
			Files []picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Files", r.Context()),
			Files:       em,
		}, template.FuncMap{
			"formatDate": func(t time.Time) string {
				return t.Format("2006-01-02")
			},
			"formatExpiration": func(et picoshare.ExpirationTime) string {
				if et == picoshare.NeverExpire {
					return "Never"
				}
				t := time.Time(et)
				delta := time.Until(t)
				return fmt.Sprintf("%s (%.0f days)", t.Format("2006-01-02"), delta.Hours()/24)
			},
			"formatFileSize": func(b int64) string {
				const unit = 1024

				if b < unit {
					return fmt.Sprintf("%d B", b)
				}
				div, exp := int64(unit), 0
				for n := b / unit; n >= unit; n /= unit {
					div *= unit
					exp++
				}
				return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "kMGTPE"[exp])
			},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileEditGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.store.GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		if err := renderTemplate(w, "file-edit.html", struct {
			commonProps
			Metadata picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Edit", r.Context()),
			Metadata:    metadata,
		}, template.FuncMap{
			"isNeverExpire": func(et picoshare.ExpirationTime) bool {
				return et == picoshare.NeverExpire
			},
			"formatExpiration": func(et picoshare.ExpirationTime) string {
				if et == picoshare.NeverExpire {
					return "Never"
				}
				return time.Time(et).Format(time.RFC3339)
			},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileConfirmDeleteGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.store.GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}
		if err := renderTemplate(w, "file-delete.html", struct {
			commonProps
			Metadata picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Delete", r.Context()),
			Metadata:    metadata,
		}, template.FuncMap{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) authGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderTemplate(w, "auth.html", struct {
			commonProps
		}{
			commonProps: makeCommonProps("PicoShare - Log in", r.Context()),
		}, template.FuncMap{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) uploadGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type expirationOption struct {
			FriendlyName string
			Expiration   time.Time
			IsDefault    bool
		}
		expirationOptions := []expirationOption{
			{"Custom", time.Time{}, false},
			{"1 day", time.Now().AddDate(0, 0, 1), false},
			{"7 days", time.Now().AddDate(0, 0, 7), false},
			{"30 days", time.Now().AddDate(0, 0, 30), true},
			{"1 year", time.Now().AddDate(1, 0, 0), false},
			{"Never", time.Time(picoshare.NeverExpire), false},
		}
		if err := renderTemplate(w, "upload.html", struct {
			commonProps
			ExpirationOptions []expirationOption
			MaxNoteLength     int
			GuestLinkMetadata picoshare.GuestLink
		}{
			commonProps:       makeCommonProps("PicoShare - Upload", r.Context()),
			MaxNoteLength:     parse.MaxFileNoteBytes,
			ExpirationOptions: expirationOptions,
		}, template.FuncMap{
			"formatExpiration": func(t time.Time) string {
				if t.IsZero() {
					return ""
				}
				return t.Format(time.RFC3339)
			}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestUploadGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		guestLinkID, err := parseGuestLinkID(mux.Vars(r)["guestLinkID"])
		if err != nil {
			log.Printf("error parsing guest link ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		gl, err := s.store.GetGuestLink(guestLinkID)
		if _, ok := err.(store.GuestLinkNotFoundError); ok {
			http.Error(w, "Invalid guest link ID", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving guest link with ID %v: %v", guestLinkID, err)
			http.Error(w, "Failed to retrieve guest link", http.StatusInternalServerError)
			return
		}

		if !gl.IsActive() {
			if err := renderTemplate(w, "guest-link-inactive.html", struct {
				commonProps
			}{
				commonProps: makeCommonProps("PicoShare - Guest Link Inactive", r.Context()),
			}, template.FuncMap{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		if err := renderTemplate(w, "upload.html", struct {
			commonProps
			ExpirationOptions []interface{}
			GuestLinkMetadata picoshare.GuestLink
		}{
			commonProps:       makeCommonProps("PicoShare - Upload", r.Context()),
			GuestLinkMetadata: gl,
		}, template.FuncMap{
			"formatExpiration": func(t time.Time) string {
				return t.Format(time.RFC3339)
			}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func makeCommonProps(title string, ctx context.Context) commonProps {
	return commonProps{
		Title:           title,
		IsAuthenticated: isAuthenticated(ctx),
		CspNonce:        cspNonce(ctx),
	}
}

//go:embed templates
var templatesFS embed.FS

func renderTemplate(w http.ResponseWriter, templateFilename string, templateVars interface{}, funcMap template.FuncMap) error {
	t := template.New(templateFilename).Funcs(funcMap)
	t = template.Must(
		t.ParseFS(
			templatesFS,
			"templates/layouts/*.html",
			"templates/partials/*.html",
			"templates/custom-elements/*.html",
			path.Join("templates/pages", templateFilename)))
	if err := t.ExecuteTemplate(w, "base", templateVars); err != nil {
		return err
	}
	return nil
}
