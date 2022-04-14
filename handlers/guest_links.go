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

const GuestLinkIDLength = 10

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
	}
}

func (s Server) guestLinksDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseGuestLinkID(mux.Vars(r)["id"])
		if err != nil {
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
		Label       string `json:"label"`
		Expiration  string `json:"expirationTime"`
		MaxFileSize *int   `json:"maxFileSize"`
		CountLimit  *int   `json:"countLimit"`
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

	sizeLimit, err := parseMaxFileSize(pr.MaxFileSize)
	if err != nil {
		return types.GuestLink{}, err
	}

	countLimit, err := parseUploadCountLimit(pr.CountLimit)
	if err != nil {
		return types.GuestLink{}, err
	}

	return types.GuestLink{
		Label:                label,
		Expires:              expiration,
		UploadSizeRemaining:  sizeLimit,
		UploadCountRemaining: countLimit,
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

func parseMaxFileSize(limitRaw *int) (*types.GuestUploadMaxFileSize, error) {
	if limitRaw == nil {
		return nil, nil
	}
	// TODO: Check more rigorously
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload size limit must be a positive number")
	}

	limit := types.GuestUploadMaxFileSize(*limitRaw)
	return &limit, nil
}

func parseUploadCountLimit(limitRaw *int) (*types.GuestUploadCountLimit, error) {
	if limitRaw == nil {
		return nil, nil
	}
	// TODO: Check more rigorously
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload count limit must be a positive number")
	}

	limit := types.GuestUploadCountLimit(*limitRaw)
	return &limit, nil
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
			return types.GuestLinkID(""), fmt.Errorf("entry ID (%s) contains invalid character: %v", s, c)
		}
	}
	return types.GuestLinkID(s), nil
}
