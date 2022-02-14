package handlers

import (
	"log"
	"net/http"
	"os"
	"path"
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
		defer file.Close()

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

		http.ServeFile(w, r, path.Join(staticRootDir, r.URL.Path))
	}
}
