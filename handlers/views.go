package handlers

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

type commonProps struct {
	Title           string
	IsAuthenticated bool
}

func (s Server) indexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.isAuthenticated(r) {
			s.uploadGet()(w, r)
			return
		}
		if err := renderTemplate(w, "index.html", struct {
			commonProps
		}{
			commonProps{
				Title:           "PicoShare",
				IsAuthenticated: s.isAuthenticated(r),
			},
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
		if err := renderTemplate(w, "guest-link-index.html", struct {
			commonProps
			GuestLinks []types.GuestLink
		}{
			commonProps: commonProps{
				Title:           "PicoShare - Guest Links",
				IsAuthenticated: s.isAuthenticated(r),
			},
			GuestLinks: links,
		}, template.FuncMap{
			"formatTime": func(t time.Time) string {
				return t.Format(time.RFC3339)
			},
			"formatSizeLimit": func(limit *types.GuestUploadMaxFileBytes) string {
				if limit == nil {
					return "Unlimited"
				}
				b := int64(*limit)
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
			"formatCountLimit": func(limit *types.GuestUploadCountLimit) string {
				if limit == nil {
					return "Unlimited"
				}
				return fmt.Sprintf("%d", int64(*limit))
			},
			"formatExpiration": func(et types.ExpirationTime) string {
				if et == types.NeverExpire {
					return "Never"
				}
				t := time.Time(et)
				delta := time.Until(t)
				return fmt.Sprintf("%s (%.0f days)", t.Format(time.RFC3339), delta.Hours()/24)
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
			commonProps: commonProps{
				Title:           "PicoShare - New Guest Link",
				IsAuthenticated: s.isAuthenticated(r),
			},
			ExpirationOptions: []expirationOption{
				{"1 day", time.Now().AddDate(0, 0, 1), false},
				{"7 days", time.Now().AddDate(0, 0, 7), false},
				{"30 days", time.Now().AddDate(0, 0, 30), false},
				{"1 year", time.Now().AddDate(1, 0, 0), false},
				{"Never", time.Time(types.NeverExpire), true},
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
			Files []types.UploadMetadata
		}{
			commonProps: commonProps{
				Title:           "PicoShare - Files",
				IsAuthenticated: s.isAuthenticated(r),
			},
			Files: em,
		}, template.FuncMap{
			"formatTime": func(t time.Time) string {
				return t.Format(time.RFC3339)
			},
			"formatExpiration": func(et types.ExpirationTime) string {
				if et == types.NeverExpire {
					return "Never"
				}
				t := time.Time(et)
				delta := time.Until(t)
				return fmt.Sprintf("%s (%.0f days)", t.Format(time.RFC3339), delta.Hours()/24)
			},
			"formatFileSize": func(b int) string {
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

func (s Server) authGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderTemplate(w, "auth.html", struct {
			commonProps
		}{
			commonProps{
				Title:           "PicoShare - Log in",
				IsAuthenticated: s.isAuthenticated(r),
			},
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
		if err := renderTemplate(w, "upload.html", struct {
			commonProps
			ExpirationOptions []expirationOption
			GuestLinkMetadata types.GuestLink
		}{
			commonProps: commonProps{
				Title:           "PicoShare - Upload",
				IsAuthenticated: s.isAuthenticated(r),
			},
			ExpirationOptions: []expirationOption{
				{"1 day", time.Now().AddDate(0, 0, 1), false},
				{"7 days", time.Now().AddDate(0, 0, 7), false},
				{"30 days", time.Now().AddDate(0, 0, 30), true},
				{"1 year", time.Now().AddDate(1, 0, 0), false},
				{"Never", time.Time(types.NeverExpire), false},
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

		// TODO: Render something different if guest link's quota is exhausted.

		type expirationOption struct {
			FriendlyName string
			Expiration   time.Time
			IsDefault    bool
		}
		if err := renderTemplate(w, "upload.html", struct {
			commonProps
			ExpirationOptions []expirationOption
			GuestLinkMetadata types.GuestLink
		}{
			commonProps: commonProps{
				Title:           "PicoShare - Upload",
				IsAuthenticated: s.isAuthenticated(r),
			},
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

func renderTemplate(w http.ResponseWriter, templateFilename string, templateVars interface{}, funcMap template.FuncMap) error {
	const templatesRootDir = "./templates"
	const baseTemplate = "base"
	const baseTemplateFilename = "base.html"
	const navbarTemplateFilename = "navbar.html"

	customElementsDir := path.Join(templatesRootDir, "custom-elements")
	customElementFiles, err := ioutil.ReadDir(customElementsDir)
	if err != nil {
		return err
	}

	customElements := []string{}
	for _, ce := range customElementFiles {
		customElements = append(customElements, path.Join(customElementsDir, ce.Name()))
	}

	templateFiles := []string{}
	templateFiles = append(templateFiles, path.Join(templatesRootDir, templateFilename))
	templateFiles = append(templateFiles, path.Join(templatesRootDir, baseTemplateFilename))
	templateFiles = append(templateFiles, path.Join(templatesRootDir, navbarTemplateFilename))
	templateFiles = append(templateFiles, customElements...)

	t := template.Must(template.New(templateFilename).Funcs(funcMap).
		ParseFiles(templateFiles...))
	if err := t.ExecuteTemplate(w, baseTemplate, templateVars); err != nil {
		return err
	}
	return nil
}
