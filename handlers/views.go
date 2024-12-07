package handlers

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/mileusna/useragent"
	"github.com/mtlynch/picoshare/v2/build"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/store"
)

//go:embed templates
var templatesFS embed.FS

type commonProps struct {
	Title           string
	IsAuthenticated bool
	CspNonce        string
}

func (s Server) indexGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r.Context()) {
			s.uploadGet()(w, r)
			return
		}
		if err := t.Execute(w, struct {
			commonProps
		}{
			commonProps: makeCommonProps("PicoShare", r.Context()),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestLinkIndexGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format(time.DateOnly)
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
			return fmt.Sprintf("%s (%.0f days%s)", t.Format(time.DateOnly), math.Abs(delta.Hours())/24, suffix)
		},
		"formatFileExpiration": func(flt picoshare.FileLifetime) string {
			if flt == picoshare.FileLifetimeInfinite {
				return "Never"
			}

			letterS := "s"
			if flt.Duration().Hours() < 25 {
				letterS = ""
			}
			return fmt.Sprintf("After %.0f day%s", math.Abs(flt.Duration().Hours()/24), letterS)
		},
		"isActive": func(gl picoshare.GuestLink) bool {
			return gl.IsActive()
		},
	}

	t := parseTemplatesWithFuncs(fns, "templates/pages/guest-link-index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := s.getDB(r).GetGuestLinks()
		if err != nil {
			log.Printf("failed to retrieve guest links: %v", err)
			http.Error(w, "Failed to retrieve guest links", http.StatusInternalServerError)
			return
		}

		sort.Slice(links, func(i, j int) bool {
			return links[i].Created.After(links[j].Created)
		})

		if err := t.Execute(w, struct {
			commonProps
			GuestLinks []picoshare.GuestLink
		}{
			commonProps: makeCommonProps("PicoShare - Guest Links", r.Context()),
			GuestLinks:  links,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestLinksNewGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatExpiration": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
	}

	t := parseTemplatesWithFuncs(fns, "templates/pages/guest-link-create.html")

	return func(w http.ResponseWriter, r *http.Request) {
		type expirationOption struct {
			FriendlyName string
			Expiration   time.Time
			IsDefault    bool
		}
		if err := t.Execute(w, struct {
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
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileIndexGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format(time.DateOnly)
		},
		"formatExpiration": func(et picoshare.ExpirationTime) string {
			if et == picoshare.NeverExpire {
				return "Never"
			}
			t := time.Time(et)
			delta := time.Until(t)
			return fmt.Sprintf("%s (%.0f days)", t.Format(time.DateOnly), delta.Hours()/24)
		},
		"formatFileSize": humanReadableFileSize,
	}

	t := parseTemplatesWithFuncs(fns, "templates/pages/file-index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		em, err := s.getDB(r).GetEntriesMetadata()
		if err != nil {
			log.Printf("failed to retrieve entries metadata: %v", err)
			http.Error(w, "failed to retrieve file index", http.StatusInternalServerError)
			return
		}
		sort.Slice(em, func(i, j int) bool {
			return em[i].Uploaded.After(em[j].Uploaded)
		})
		if err := t.Execute(w, struct {
			commonProps
			Files []picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Files", r.Context()),
			Files:       em,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileEditGet() http.HandlerFunc {
	fns := template.FuncMap{
		"isNeverExpire": func(et picoshare.ExpirationTime) bool {
			return et == picoshare.NeverExpire
		},
		"formatExpiration": func(et picoshare.ExpirationTime) string {
			if et == picoshare.NeverExpire {
				return "Never"
			}
			return time.Time(et).Format(time.RFC3339)
		},
	}

	t := parseTemplatesWithFuncs(fns,
		"templates/custom-elements/expiration-picker.html",
		"templates/pages/file-edit.html")

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.getDB(r).GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		if err := t.Execute(w, struct {
			commonProps
			Metadata picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Edit", r.Context()),
			Metadata:    metadata,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileInfoGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatExpiration": func(et picoshare.ExpirationTime) string {
			if et == picoshare.NeverExpire {
				return "Never"
			}
			t := time.Time(et)
			delta := time.Until(t)
			return fmt.Sprintf("%s (%.0f days)", t.Format(time.DateOnly), delta.Hours()/24)
		},
		"formatTimestamp": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
		"formatFileSize": humanReadableFileSize,
	}

	t := parseTemplatesWithFuncs(
		fns,
		"templates/custom-elements/upload-link-box.html",
		"templates/custom-elements/upload-links.html",
		"templates/pages/file-info.html")

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.getDB(r).GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		downloads, err := s.getDB(r).GetEntryDownloads(id)
		if err != nil {
			log.Printf("error retrieving downloads for id %v: %v", id, err)
			http.Error(w, "failed to retrieve downloads", http.StatusInternalServerError)
			return
		}

		if err := t.Execute(w, struct {
			commonProps
			Metadata      picoshare.UploadMetadata
			DownloadCount int
		}{
			commonProps:   makeCommonProps("PicoShare - File Information", r.Context()),
			Metadata:      metadata,
			DownloadCount: len(downloads),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileDownloadsGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatDownloadIndex": func(i, total int) int {
			return total - i
		},
		"formatDownloadTime": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
	}
	t := parseTemplatesWithFuncs(fns, "templates/pages/file-downloads.html")

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		db := s.getDB(r)

		metadata, err := db.GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		downloads, err := db.GetEntryDownloads(id)
		if err != nil {
			log.Printf("error retrieving downloads for id %v: %v", id, err)
			http.Error(w, "failed to retrieve downloads", http.StatusInternalServerError)
			return
		}

		// Convert raw downloads to display-friendly information.
		type downloadRecord struct {
			Time     time.Time
			ClientIP string
			Browser  string
			Platform string
		}
		records := make([]downloadRecord, len(downloads))
		for i, d := range downloads {
			agent := useragent.Parse(d.UserAgent)
			records[i] = downloadRecord{
				Time:     d.Time,
				ClientIP: d.ClientIP,
				Browser:  agent.Name,
				Platform: agent.OS,
			}
		}

		if err := t.Execute(w, struct {
			commonProps
			Metadata  picoshare.UploadMetadata
			Downloads []downloadRecord
		}{
			commonProps: makeCommonProps("PicoShare - Downloads", r.Context()),
			Metadata:    metadata,
			Downloads:   records,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) fileConfirmDeleteGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/file-delete.html")

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.getDB(r).GetEntryMetadata(id)
		if _, ok := err.(store.EntryNotFoundError); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, struct {
			commonProps
			Metadata picoshare.UploadMetadata
		}{
			commonProps: makeCommonProps("PicoShare - Delete", r.Context()),
			Metadata:    metadata,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) authGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/auth.html")

	return func(w http.ResponseWriter, r *http.Request) {
		if err := t.Execute(w, struct {
			commonProps
		}{
			commonProps: makeCommonProps("PicoShare - Log in", r.Context()),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) uploadGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatExpiration": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format(time.RFC3339)
		},
	}

	t := parseTemplatesWithFuncs(
		fns,
		"templates/custom-elements/expiration-picker.html",
		"templates/custom-elements/upload-link-box.html",
		"templates/custom-elements/upload-links.html",
		"templates/pages/upload.html")

	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := s.getDB(r).ReadSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read settings from database: %v", err), http.StatusInternalServerError)
			return
		}
		type lifetimeOption struct {
			Lifetime  picoshare.FileLifetime
			IsDefault bool
		}
		lifetimeOptions := []lifetimeOption{
			{picoshare.NewFileLifetimeInDays(1), false},
			{picoshare.NewFileLifetimeInDays(7), false},
			{picoshare.NewFileLifetimeInDays(30), false},
			{picoshare.NewFileLifetimeInYears(1), false},
			{picoshare.FileLifetimeInfinite, false},
		}

		defaultIsBuiltIn := false
		for i, lto := range lifetimeOptions {
			if lto.Lifetime.Equal(settings.DefaultFileLifetime) {
				lifetimeOptions[i].IsDefault = true
				defaultIsBuiltIn = true
			}
		}
		// If the default isn't one of the built-in options, add it and sort the
		// list.
		if !defaultIsBuiltIn {
			lifetimeOptions = append(lifetimeOptions, lifetimeOption{settings.DefaultFileLifetime, true})
			sort.Slice(lifetimeOptions, func(i, j int) bool {
				return lifetimeOptions[i].Lifetime.Duration() < lifetimeOptions[j].Lifetime.Duration()
			})
		}

		type expirationOption struct {
			FriendlyName string
			Expiration   time.Time
			IsDefault    bool
		}
		expirationOptions := []expirationOption{}
		for _, lto := range lifetimeOptions {
			friendlyName := lto.Lifetime.FriendlyName()
			expiration := time.Now().Add(lto.Lifetime.Duration())
			if lto.Lifetime.Equal(picoshare.FileLifetimeInfinite) {
				friendlyName = "Never"
				expiration = time.Time(picoshare.NeverExpire)
			}
			expirationOptions = append(expirationOptions, expirationOption{
				FriendlyName: friendlyName,
				Expiration:   expiration,
				IsDefault:    lto.IsDefault,
			})
		}

		expirationOptions = append(expirationOptions, expirationOption{"Custom", time.Time{}, false})

		if err := t.Execute(w, struct {
			commonProps
			ExpirationOptions []expirationOption
			MaxNoteLength     int
			GuestLinkMetadata picoshare.GuestLink
		}{
			commonProps:       makeCommonProps("PicoShare - Upload", r.Context()),
			MaxNoteLength:     parse.MaxFileNoteBytes,
			ExpirationOptions: expirationOptions,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestUploadGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatExpiration": func(t time.Time) string {
			return t.Format(time.RFC3339)
		}}

	t := parseTemplatesWithFuncs(
		fns,
		"templates/custom-elements/expiration-picker.html",
		"templates/custom-elements/upload-link-box.html",
		"templates/custom-elements/upload-links.html",
		"templates/pages/upload.html")

	tInactive := parseTemplates("templates/pages/guest-link-inactive.html")

	return func(w http.ResponseWriter, r *http.Request) {
		guestLinkID, err := parseGuestLinkID(mux.Vars(r)["guestLinkID"])
		if err != nil {
			log.Printf("error parsing guest link ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		gl, err := s.getDB(r).GetGuestLink(guestLinkID)
		if _, ok := err.(store.GuestLinkNotFoundError); ok {
			http.Error(w, "Invalid guest link ID", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving guest link with ID %v: %v", guestLinkID, err)
			http.Error(w, "Failed to retrieve guest link", http.StatusInternalServerError)
			return
		}

		if !gl.IsActive() {
			if err := tInactive.Execute(w, struct {
				commonProps
			}{
				commonProps: makeCommonProps("PicoShare - Guest Link Inactive", r.Context()),
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		if err := t.Execute(w, struct {
			commonProps
			ExpirationOptions []interface{}
			GuestLinkMetadata picoshare.GuestLink
		}{
			commonProps:       makeCommonProps("PicoShare - Upload", r.Context()),
			GuestLinkMetadata: gl,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) settingsGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/settings.html")

	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := s.getDB(r).ReadSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read settings from database: %v", err), http.StatusInternalServerError)
			return
		}
		var defaultExpiration uint16
		var expirationTimeUnit string
		defaultNeverExpire := settings.DefaultFileLifetime.Equal(picoshare.FileLifetimeInfinite)
		if defaultNeverExpire {
			defaultExpiration = 30
			expirationTimeUnit = "days"
		} else {
			defaultExpiration = settings.DefaultFileLifetime.Days()
			expirationTimeUnit = "days"
			if settings.DefaultFileLifetime.IsYearBoundary() {
				defaultExpiration = settings.DefaultFileLifetime.Years()
				expirationTimeUnit = "years"
			}
		}

		if err := t.Execute(w, struct {
			commonProps
			DefaultExpiration  uint16
			ExpirationTimeUnit string
			DefaultNeverExpire bool
		}{
			commonProps:        makeCommonProps("PicoShare - Settings", r.Context()),
			DefaultExpiration:  defaultExpiration,
			ExpirationTimeUnit: expirationTimeUnit,
			DefaultNeverExpire: defaultNeverExpire,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) systemInformationGet() http.HandlerFunc {
	fns := template.FuncMap{
		"formatFileSize": humanReadableFileSize,
		"percentage": func(part, total uint64) string {
			return fmt.Sprintf("%.0f%%", 100.0*(float64(part)/float64(total)))
		},
	}
	t := parseTemplatesWithFuncs(fns, "templates/pages/system-information.html")

	return func(w http.ResponseWriter, r *http.Request) {
		spaceUsage, err := s.spaceChecker.Check()
		if err != nil {
			log.Printf("error checking available space: %v", err)
			http.Error(w, fmt.Sprintf("failed to check available space: %v", err), http.StatusInternalServerError)
			return
		}

		if err := t.Execute(w, struct {
			commonProps
			TotalServingBytes uint64
			DatabaseFileBytes uint64
			UsedBytes         uint64
			TotalBytes        uint64
			BuildTime         time.Time
			Version           string
		}{
			commonProps:       makeCommonProps("PicoShare - System Information", r.Context()),
			TotalServingBytes: spaceUsage.TotalServingBytes,
			DatabaseFileBytes: spaceUsage.DatabaseFileSize,
			UsedBytes:         spaceUsage.FileSystemUsedBytes,
			TotalBytes:        spaceUsage.FileSystemTotalBytes,
			BuildTime:         build.Time(),
			Version:           build.Version,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func humanReadableFileSize(b uint64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func makeCommonProps(title string, ctx context.Context) commonProps {
	return commonProps{
		Title:           title,
		IsAuthenticated: isAuthenticated(ctx),
		CspNonce:        cspNonce(ctx),
	}
}

func parseTemplates(templatePaths ...string) *template.Template {
	return parseTemplatesWithFuncs(template.FuncMap{}, templatePaths...)
}

func parseTemplatesWithFuncs(fns template.FuncMap, templatePaths ...string) *template.Template {
	return template.Must(
		template.New("base.html").
			Funcs(fns).
			ParseFS(
				templatesFS,
				append(
					[]string{
						"templates/layouts/base.html",
						"templates/partials/navbar.html",
						"templates/custom-elements/snackbar-notifications.html",
					},
					templatePaths...)...))
}
