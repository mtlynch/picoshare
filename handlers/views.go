package handlers

import (
	"net/http"
	"path"
	"text/template"
)

type commonProps struct {
	Title      string
	IsLoggedIn bool
}

func indexGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderTemplate(w, "index.html", struct {
			commonProps
		}{
			commonProps{
				Title: "PicoShare",
			},
		}, template.FuncMap{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func authGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderTemplate(w, "auth.html", struct {
			commonProps
		}{
			commonProps{
				Title: "PicoShare - Authenticate",
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

	t := template.Must(template.New(templateFilename).Funcs(funcMap).
		ParseFiles(
			path.Join(templatesRootDir, templateFilename),
			path.Join(templatesRootDir, baseTemplateFilename),
			path.Join(templatesRootDir, navbarTemplateFilename),
		))
	if err := t.ExecuteTemplate(w, baseTemplate, templateVars); err != nil {
		return err
	}
	return nil
}
