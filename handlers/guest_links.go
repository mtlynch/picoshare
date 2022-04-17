package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/types"
)

const GuestLinkIDLength = 16

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

		gl.Created = time.Now()

		if err := s.store.InsertGuestLink(gl); err != nil {
			log.Printf("failed to save guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save guest link: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(GuestLinkPostResponse{
			ID: string(gl.ID),
		}); err != nil {
			panic(err)
		}
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

		if err := s.store.DeleteGuestLink(id); err != nil {
			log.Printf("failed to delete guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete guest link: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func guestLinkFromRequest(r *http.Request) (types.GuestLink, error) {
	type preferencesRequest struct {
		Label          string  `json:"label"`
		Expiration     string  `json:"expirationTime"`
		MaxFileBytes   *uint64 `json:"maxFileBytes"`
		MaxFileUploads *int    `json:"maxFileUploads"`
	}
	var pr preferencesRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&pr)
	if err != nil {
		log.Printf("failed to decode JSON request: %v", err)
		return types.GuestLink{}, err
	}

	label, err := parseLabel(pr.Label)
	if err != nil {
		return types.GuestLink{}, err
	}

	expiration, err := parseExpiration(pr.Expiration)
	if err != nil {
		return types.GuestLink{}, err
	}

	maxFileBytes, err := parseMaxFileBytes(pr.MaxFileBytes)
	if err != nil {
		return types.GuestLink{}, err
	}

	maxFileUploads, err := parseUploadCountLimit(pr.MaxFileUploads)
	if err != nil {
		return types.GuestLink{}, err
	}

	return types.GuestLink{
		Label:          label,
		Expires:        expiration,
		MaxFileBytes:   maxFileBytes,
		MaxFileUploads: maxFileUploads,
	}, nil
}

func parseLabel(label string) (types.GuestLinkLabel, error) {
	// Arbitrary limit to prevent too-long labels
	limit := 200
	if len(label) > limit {
		return types.GuestLinkLabel(""), fmt.Errorf("label too long - limit %d characters", limit)
	}

	return types.GuestLinkLabel(label), nil
}

func parseMaxFileBytes(limitRaw *uint64) (types.GuestUploadMaxFileBytes, error) {
	if limitRaw == nil {
		return types.GuestUploadUnlimitedFileSize, nil
	}
	// TODO: Check more rigorously
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload size limit must be a positive number")
	}

	return types.GuestUploadMaxFileBytes(limitRaw), nil
}

func parseUploadCountLimit(limitRaw *int) (types.GuestUploadCountLimit, error) {
	if limitRaw == nil {
		return types.GuestUploadUnlimitedFileUploads, nil
	}
	// TODO: Check more rigorously
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload count limit must be a positive number")
	}

	return types.GuestUploadCountLimit(limitRaw), nil
}

func generateGuestLinkID() types.GuestLinkID {
	return types.GuestLinkID(random.String(GuestLinkIDLength, guestLinkIDCharacters))
}

func parseGuestLinkID(s string) (types.GuestLinkID, error) {
	if len(s) != GuestLinkIDLength {
		return types.GuestLinkID(""), fmt.Errorf("guest link ID (%v) has invalid length: got %d, want %d", s, len(s), GuestLinkIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range guestLinkIDCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return types.GuestLinkID(""), fmt.Errorf("entry ID (%s) contains invalid character: %s", s, string(c))
		}
	}
	return types.GuestLinkID(s), nil
}
