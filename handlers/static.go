package handlers

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"
)

//go:embed static
var staticFS embed.FS

// serveStaticResource serves any static file under the ./handlers/static
// directory.
func serveStaticResource() http.HandlerFunc {
	fSys, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	server := http.FileServer(http.FS(fSys))

	return func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers
		if mt, ok := lastModTime(staticFS, r.URL.Path); ok {
			etag := "\"" + strconv.FormatInt(mt.UnixMilli(), 10) + "\""
			w.Header().Set("Etag", etag)
			w.Header().Set("Cache-Control", "max-age=3600")
		}

		server.ServeHTTP(w, r)
	}
}

func lastModTime(fs fs.FS, path string) (time.Time, bool) {
	file, err := fs.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file handle for %s: %v", path, err)
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return time.Time{}, false
	}
	return stat.ModTime(), true
}
