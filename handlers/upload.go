package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/types"
)

const (
	FileLifetime  = 7 * 24 * time.Hour
	EntryIDLength = 10
)

// Omit visually similar characters (I,l,1), (0,O)
var entryIDCharacters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

type (
	EntryPostResponse struct {
		ID string `json:"id"`
	}

	fileUpload struct {
		Reader      io.Reader
		Filename    types.Filename
		Note        types.FileNote
		ContentType types.ContentType
	}
)

func (s Server) entryPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expiration, err := parseExpirationFromRequest(r)
		if err != nil {
			log.Printf("invalid expiration URL parameter: %v", err)
			http.Error(w, fmt.Sprintf("Invalid expiration URL parameter: %v", err), http.StatusBadRequest)
			return
		}

		uploadedFile, err := fileFromRequest(w, r)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		id := generateEntryID()
		err = s.store.InsertEntry(uploadedFile.Reader,
			types.UploadMetadata{
				Filename:    uploadedFile.Filename,
				Note:        uploadedFile.Note,
				ContentType: uploadedFile.ContentType,
				ID:          id,
				Uploaded:    time.Now(),
				Expires:     types.ExpirationTime(expiration),
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

func (s Server) guestEntryPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		guestLinkID, err := parseGuestLinkID(mux.Vars(r)["guestLinkID"])
		if err != nil {
			log.Printf("error parsing guest link ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		gl, err := s.store.GetGuestLink(guestLinkID)
		if _, ok := err.(store.GuestLinkNotFoundError); ok {
			http.Error(w, "Invalid guest link ID", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving guest link with ID %v: %v", guestLinkID, err)
			http.Error(w, "Failed to retrieve guest link", http.StatusInternalServerError)
			return
		}

		if !gl.IsActive() {
			http.Error(w, "Guest link is no longer active", http.StatusUnauthorized)
		}

		if gl.MaxFileBytes != types.GuestUploadUnlimitedFileSize {
			// We technically allow slightly less than the user specified because
			// other fields in the request take up some space, but it's a difference
			// of only a few hundred bytes.
			r.Body = http.MaxBytesReader(w, r.Body, int64(*gl.MaxFileBytes))
		}

		uploadedFile, err := fileFromRequest(w, r)
		if err != nil {
			log.Printf("error reading body: %v", err)
			http.Error(w, fmt.Sprintf("can't read request body: %s", err), http.StatusBadRequest)
			return
		}

		if uploadedFile.Note.Value != nil {
			http.Error(w, "Guest uploads cannot have file notes", http.StatusBadRequest)
			return
		}

		id := generateEntryID()
		err = s.store.InsertEntry(uploadedFile.Reader,
			types.UploadMetadata{
				Filename:    uploadedFile.Filename,
				ContentType: uploadedFile.ContentType,
				ID:          id,
				GuestLinkID: guestLinkID,
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

func generateEntryID() types.EntryID {
	return types.EntryID(random.String(EntryIDLength, entryIDCharacters))
}

func parseEntryID(s string) (types.EntryID, error) {
	if len(s) != EntryIDLength {
		return types.EntryID(""), fmt.Errorf("entry ID (%v) has invalid length: got %d, want %d", s, len(s), EntryIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range entryIDCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return types.EntryID(""), fmt.Errorf("entry ID (%s) contains invalid character: %v", s, c)
		}
	}
	return types.EntryID(s), nil
}

func fileFromRequest(w http.ResponseWriter, r *http.Request) (fileUpload, error) {
	// We're intentionally not limiting the size of the request because we assume
	// the the uploading user is trusted, so they can upload files of any size
	// they want.
	r.ParseMultipartForm(32 << 20)
	reader, metadata, err := r.FormFile("file")
	if err != nil {
		return fileUpload{}, err
	}

	if metadata.Size == 0 {
		return fileUpload{}, errors.New("file is empty")
	}

	filename, err := parse.Filename(metadata.Filename)
	if err != nil {
		return fileUpload{}, err
	}

	contentType, err := parseContentType(metadata.Header.Get("Content-Type"))
	if err != nil {
		return fileUpload{}, err
	}

	note, err := parse.FileNote(r.FormValue("note"))
	if err != nil {
		return fileUpload{}, err
	}

	return fileUpload{
		Reader:      reader,
		Filename:    filename,
		Note:        note,
		ContentType: contentType,
	}, nil
}

func parseContentType(s string) (types.ContentType, error) {
	// The content type header is fairly open-ended, so we're liberal in what
	// values we accept.
	return types.ContentType(s), nil
}

func parseExpirationFromRequest(r *http.Request) (types.ExpirationTime, error) {
	expirationRaw, ok := r.URL.Query()["expiration"]
	if !ok {
		return types.ExpirationTime{}, errors.New("missing required URL parameter: expiration")
	}
	if len(expirationRaw) <= 0 {
		return types.ExpirationTime{}, errors.New("missing required URL parameter: expiration")
	}
	return parseExpiration(expirationRaw[0])
}

func parseExpiration(expirationRaw string) (types.ExpirationTime, error) {
	expiration, err := time.Parse(time.RFC3339, expirationRaw)
	if err != nil {
		log.Printf("invalid expiration URL parameter: %v -> %v", expirationRaw, err)
		return types.ExpirationTime{}, errors.New("invalid expiration URL parameter")
	}

	if time.Until(expiration) < (time.Hour * 1) {
		return types.ExpirationTime{}, errors.New("expire time must be at least one hour in the future")
	}

	return types.ExpirationTime(expiration), nil
}
