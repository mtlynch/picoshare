package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// clearPost completely clears the database (for testing only).
// This endpoint is authentication-protected to prevent abuse.
func (s *Server) clearPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.ClearAll(); err != nil {
			log.Printf("database clear failed: %v", err)
			http.Error(w, fmt.Sprintf("database clear failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Clean up /tmp to prevent disk space issues from multipart temp files
		// Only delete multipart-* files, leave everything else alone
		tmpDir, err := os.Open("/tmp")
		if err == nil {
			defer tmpDir.Close()
			entries, err := tmpDir.Readdirnames(-1)
			if err == nil {
				for _, entry := range entries {
					// Only delete multipart temp files
					if strings.HasPrefix(entry, "multipart-") {
						_ = os.RemoveAll(filepath.Join("/tmp", entry))
					}
				}
			}
		}

		log.Printf("database and /tmp cleared successfully")
	}
}
