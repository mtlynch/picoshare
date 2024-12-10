package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/random"
)

const (
	GuestLinkIDLength         = 16
	GuestLinkByteLimitMinimum = 1024 * 1024
)

type GuestLinkPostResponse struct {
	ID string `json:"id"`
}

// Omit visually similar characters (I,l,1), (0,O)
var guestLinkIDCharacters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func (s Server) guestLinksPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gl, err := guestLinkFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		gl.ID = generateGuestLinkID()
		gl.Created = s.clock.Now()

		if err := s.getDB(r).InsertGuestLink(gl); err != nil {
			log.Printf("failed to save guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save guest link: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, GuestLinkPostResponse{ID: string(gl.ID)})
	}
}

func (s Server) guestLinksDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseGuestLinkID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("failed to parse guest link ID %s: %v", mux.Vars(r)["id"], err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		if err := s.getDB(r).DeleteGuestLink(id); err != nil {
			log.Printf("failed to delete guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete guest link: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func guestLinkFromRequest(r *http.Request) (picoshare.GuestLink, error) {
	var payload struct {
		Label          string  `json:"label"`
		UrlExpiration  string  `json:"urlExpirationTime"`
		FileExpiration string  `json:"fileLifetime"`
		MaxFileBytes   *uint64 `json:"maxFileBytes"`
		MaxFileUploads *int    `json:"maxFileUploads"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("failed to decode JSON request: %v", err)
		return picoshare.GuestLink{}, err
	}

	label, err := parse.GuestLinkLabel(payload.Label)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	urlExpiration, err := parse.Expiration(payload.UrlExpiration)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	fileExpiration, err := parse.FileLifetimeFromString(payload.FileExpiration)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	maxFileBytes, err := parseMaxFileBytes(payload.MaxFileBytes)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	maxFileUploads, err := parseUploadCountLimit(payload.MaxFileUploads)
	if err != nil {
		return picoshare.GuestLink{}, err
	}

	return picoshare.GuestLink{
		Label:          label,
		UrlExpires:     urlExpiration,
		FileLifetime:   fileExpiration,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
	}, nil
}

func parseMaxFileBytes(limitRaw *uint64) (picoshare.GuestUploadMaxFileBytes, error) {
	if limitRaw == nil {
		return picoshare.GuestUploadUnlimitedFileSize, nil
	}
	if *limitRaw < GuestLinkByteLimitMinimum {
		return nil, fmt.Errorf("guest upload size limit must be at at least %d bytes", GuestLinkByteLimitMinimum)
	}

	return picoshare.GuestUploadMaxFileBytes(limitRaw), nil
}

func parseUploadCountLimit(limitRaw *int) (picoshare.GuestUploadCountLimit, error) {
	if limitRaw == nil {
		return picoshare.GuestUploadUnlimitedFileUploads, nil
	}
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload count limit must be a positive number")
	}

	return picoshare.GuestUploadCountLimit(limitRaw), nil
}

func generateGuestLinkID() picoshare.GuestLinkID {
	return picoshare.GuestLinkID(random.String(GuestLinkIDLength, guestLinkIDCharacters))
}

func parseGuestLinkID(s string) (picoshare.GuestLinkID, error) {
	if len(s) != GuestLinkIDLength {
		return picoshare.GuestLinkID(""), fmt.Errorf("guest link ID (%v) has invalid length: got %d, want %d", s, len(s), GuestLinkIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range guestLinkIDCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return picoshare.GuestLinkID(""), fmt.Errorf("entry ID (%s) contains invalid character: %s", s, string(c))
		}
	}
	return picoshare.GuestLinkID(s), nil
}
