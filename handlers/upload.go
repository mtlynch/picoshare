package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/types"
)

const (
	MaxFilenameLen = 100
	FileLifetime   = 7 * 24 * time.Hour
	EntryIDLength  = 14
)

var (
	maxUploadBytes = megabytesToBytes(750)
	idCharacters   = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")
)

type EntryPostResponse struct {
	ID string `json:"id"`
}

func (s Server) entryPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expiration, err := parseExpiration(r)
		if err != nil {
			log.Printf("invalid expiration URL parameter: %v", err)
			http.Error(w, "invalid expiration URL parameter", http.StatusBadRequest)
			return
		}

		reader, filename, err := fileFromRequest(w, r)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		id := generateEntryID()
		err = s.store.InsertEntry(reader,
			types.UploadMetadata{
				Filename: filename,
				ID:       id,
				Uploaded: time.Now(),
				Expires:  types.ExpirationTime(expiration),
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
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
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
	if len(s) > MaxFilenameLen {
		return types.Filename(""), errors.New("filename too long")
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return types.Filename(""), errors.New("illegal filename")
	}
	if strings.ContainsAny(s, "\\") {
		return types.Filename(""), errors.New("illegal characters in filename")
	}
	return types.Filename(s), nil
}

func parseExpiration(r *http.Request) (types.ExpirationTime, error) {
	expirationRaw, ok := r.URL.Query()["expiration"]
	if !ok {
		return types.ExpirationTime{}, errors.New("missing required URL parameter: expiration")
	}
	if len(expirationRaw) <= 0 {
		return types.ExpirationTime{}, errors.New("missing required URL parameter: expiration")
	}
	expiration, err := time.Parse(time.RFC3339, expirationRaw[0])
	if err != nil {
		log.Printf("invalid expiration URL parameter: %v -> %v", expirationRaw, err)
		return types.ExpirationTime{}, errors.New("invalid expiration URL parameter")
	}

	if time.Until(expiration) < (time.Hour * 1) {
		return types.ExpirationTime{}, errors.New("expire time must be at least one hour in the future")
	}

	return types.ExpirationTime(expiration), nil
}

func megabytesToBytes(gb int) int64 {
	return int64(gb) * 1000 * 1000
}
