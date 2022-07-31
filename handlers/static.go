package handlers

import (
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
)

// serveStaticResource serves any static file under the ./static directory.
func serveStaticResource() http.HandlerFunc {
	const staticRootDir = "./static"
	return func(w http.ResponseWriter, r *http.Request) {
		fs := http.Dir(staticRootDir)
		file, err := fs.Open(r.URL.Path)
		if os.IsNotExist(err) {
			log.Printf("%s does not exist on the file system: %s", r.URL.Path, err)
			http.Error(w, "Failed to find file: "+r.URL.Path, http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("Failed to retrieve the file %s from the file system: %s", r.URL.Path, err)
			http.Error(w, "Failed to find file: "+r.URL.Path, http.StatusNotFound)
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("failed to close file handle for %s: %v", r.URL.Path, err)
			}
		}()

		stat, err := file.Stat()
		if err != nil {
			log.Printf("Failed to retrieve the information of %s from the file system: %s", r.URL.Path, err)
			http.Error(w, "Failed to serve: "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			log.Printf("%s is a directory", r.URL.Path)
			http.Error(w, "Failed to find file: "+r.URL.Path, http.StatusNotFound)
			return
		}

		// Set cache headers
		etag := "\"" + strconv.FormatInt(stat.ModTime().UnixMilli(), 10) + "\""
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=3600")

		http.ServeFile(w, r, path.Join(staticRootDir, r.URL.Path))
	}
}
