package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/types"
)

const MaxUploadBytes = 100 * 1000 * 1000

func (s Server) entryGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idRaw := mux.Vars(r)["id"]

		// TODO: Parse ID for real
		id := types.EntryID(idRaw)

		entry, err := s.store.GetEntry(id)
		if err != nil {
			log.Printf("Error retrieving entry with id %s: %v", id, err)
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		w.Write(entry.Data)
	}
}
func (s Server) entryPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, err := fileFromRequest(w, r)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		id := generateEntryId()
		err = s.store.InsertEntry(id, types.UploadEntry{
			Data: data,
		})
		if err != nil {
			log.Printf("failed to save entry: %v", err)
			http.Error(w, "can't save entry", http.StatusInternalServerError)
			return
		}
		log.Printf("saved entry of %d bytes", len(data))

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(struct {
			ID string
		}{
			ID: string(id),
		}); err != nil {
			panic(err)
		}
	}
}

func generateEntryId() types.EntryID {
	return types.EntryID(random.String(14))
}

func fileFromRequest(w http.ResponseWriter, r *http.Request) (io.Reader, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes)
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	return file, nil
}
