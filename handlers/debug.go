package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/types"
	"github.com/pkg/profile"
)

type randomDataReader struct {
	bytesRemaining int
}

type stopper interface {
	Stop()
}

var memProfile stopper

func (rdr *randomDataReader) Read(p []byte) (n int, err error) {
	if len(p) <= rdr.bytesRemaining {
		rdr.bytesRemaining -= len(p)
		return rand.Reader.Read(p)
	}
	copy(p, random.Bytes(rdr.bytesRemaining))
	bytesRead := rdr.bytesRemaining
	rdr.bytesRemaining = 0

	return bytesRead, io.EOF
}

func (s Server) debugWriteData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		size, err := strconv.Atoi(mux.Vars(r)["size"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("writing %d random bytes to data store", size)
		reader := randomDataReader{size}
		note, _ := parse.FileNote("dummy note")

		id := generateEntryID()
		err = s.store.InsertEntry(&reader,
			types.UploadMetadata{
				Filename:    "dummy file name",
				Note:        note,
				ContentType: "application/octet-stream",
				ID:          id,
				Uploaded:    time.Now(),
				Expires:     types.NeverExpire,
			})
		if err != nil {
			log.Printf("failed to save entry: %v", err)
			http.Error(w, "can't save entry", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(EntryPostResponse{
			ID: string(id),
		}); err != nil {
			panic(err)
		}
	}
}

func (s Server) debugMemory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "PicoShare's view of its RAM allocations:\r\n")
		fmt.Fprintf(w, "Alloc:      %s\r\n", formatFileSize(m.Alloc))
		fmt.Fprintf(w, "TotalAlloc: %s\r\n", formatFileSize(m.TotalAlloc))
		fmt.Fprintf(w, "NumGC:      %d\r\n", m.NumGC)
		f, err := os.Open("/proc/meminfo")
		if err != nil {
			log.Printf("failed to read /proc/meminfo: %v", err)
			return
		}
		fmt.Fprint(w, "\r\n/proc/meminfo:\r\n")
		io.Copy(w, f)
	}
}

func (s Server) debugProfileStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memProfile = profile.Start(profile.MemProfile)

	}
}

func (s Server) debugProfileStop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memProfile.Stop()
	}
}

func formatFileSize(b uint64) string {
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
}
