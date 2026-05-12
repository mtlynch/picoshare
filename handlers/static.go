package handlers

import (
	"embed"
	"io/fs"
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
	// Because we embed static files in the Go binary, we just use the first time
	// this function runs as the last modification time for caching headers.
	lastModificationTime := time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers
		etag := "\"" + strconv.FormatInt(lastModificationTime.UnixMilli(), 10) + "\""
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=3600")

		server.ServeHTTP(w, r)
	}
}
