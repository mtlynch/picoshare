package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/types"
)

const (
	MaxUploadBytes = 100 * 1000 * 1000
	FileLifetime   = 24 * 7 * time.Hour
	EntryIDLength  = 14
)

var idCharacters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

var entries = map[types.EntryID]types.UploadEntry{}

func (s Server) entryGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		entry, ok := entries[id]
		if !ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		}

		if entry.Filename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, entry.Filename))
		}
		w.Write(entry.Data)
	}
}
func (s Server) entryPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, filename, err := fileFromRequest(w, r)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		id := generateEntryID()
		entries[id] = types.UploadEntry{
			Filename: filename,
			Data:     data,
			Uploaded: time.Now(),
			Expires:  time.Now().Add(FileLifetime),
		}
		if err != nil {
			log.Printf("failed to save entry: %v", err)
			http.Error(w, "can't save entry", http.StatusInternalServerError)
			return
		}

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

func generateEntryID() types.EntryID {
	return types.EntryID(random.String(EntryIDLength, idCharacters))
}

func parseEntryID(s string) (types.EntryID, error) {
	if len(s) != EntryIDLength {
		return types.EntryID(""), fmt.Errorf("entry ID (%v) has invalid length: got %d, want %d", s, len(s), EntryIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range idCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return types.EntryID(""), fmt.Errorf("entry ID (%s) contains invalid character: %v", s, c)
		}
	}
	return types.EntryID(s), nil
}

func fileFromRequest(w http.ResponseWriter, r *http.Request) (io.Reader, types.Filename, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes)
	r.ParseMultipartForm(32 << 20)
	file, metadata, err := r.FormFile("file")
	if err != nil {
		return nil, "", err
	}

	filename, err := parseFilename(metadata.Filename)
	if err != nil {
		return nil, "", err
	}

	return file, filename, nil
}

func parseFilename(s string) (types.Filename, error) {
	if len(s) > 100 {
		return types.Filename(""), errors.New("filename too long")
	}
	for _, c := range s {
		if c == '/' || c == '\\' {
			return types.Filename(""), errors.New("illegal characters in filename")
		}
	}
	return types.Filename(s), nil
}