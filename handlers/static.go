package handlers

import (
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

// serveStaticResource serves any static file under the ./static directory.
func serveStaticResource() http.HandlerFunc {
	const staticRootDir = "./static"
	fs := http.FileServer(http.Dir(staticRootDir))

	return func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers
		if mt, ok := lastModTime(path.Join(staticRootDir, r.URL.Path)); ok {
			etag := "\"" + strconv.FormatInt(mt.UnixMilli(), 10) + "\""
			w.Header().Set("Etag", etag)
			w.Header().Set("Cache-Control", "max-age=3600")
		}

		fs.ServeHTTP(w, r)
	}
}

func lastModTime(path string) (time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return time.Time{}, false
	}
	return stat.ModTime(), true
}
