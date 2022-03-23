package handlers

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"time"

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

func (s Server) fileIndexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		em, err := s.store.GetEntriesMetadata()
		if err != nil {
			log.Printf("failed to retrieve entries metadata: %v", err)
			http.Error(w, "failed to retrieve file index", http.StatusInternalServerError)
			return
		}
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
				t := time.Time(et)
				// zero time is interpreted as "never"
				// return a meaningful expiration string instead of -XXXX years
				if t.IsZero() {
					return "N/A"
				}
				delta := time.Until(t)
				return fmt.Sprintf("%s (%.0f days)", t.Format(time.RFC3339), delta.Hours()/24)
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
		if err := renderTemplate(w, "upload.html", struct {
			commonProps
		}{
			commonProps{
				Title:           "PicoShare - Upload",
				IsAuthenticated: s.isAuthenticated(r),
			},
		}, template.FuncMap{}); err != nil {
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
	templateFiles = append(templateFiles, path.Join(templatesRootDir, navbarTemplateFilename))
	templateFiles = append(templateFiles, customElements...)

	t := template.Must(template.New(templateFilename).Funcs(funcMap).
		ParseFiles(templateFiles...))
	if err := t.ExecuteTemplate(w, baseTemplate, templateVars); err != nil {
		return err
	}
	return nil
}
