package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mtlynch/picoshare/v2/types"
)

func (s Server) guestLinksPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		gl, err := guestLinkFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// TODO: Create a better ID
		gl.ID = types.GuestLinkID(time.Now().Format(time.RFC3339Nano))

		gl.Created = time.Now()

		if err := s.store.InsertGuestLink(gl); err != nil {
			log.Printf("failed to save guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save guest link: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func guestLinkFromRequest(r *http.Request) (types.GuestLink, error) {
	type preferencesRequest struct {
		Label      string `json:"label"`
		Expiration string `json:"expirationTime"`
		SizeLimit  *int   `json:"sizeLimit"`
		CountLimit *int   `json:"countLimit"`
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

	sizeLimit, err := parseSizeLimit(pr.SizeLimit)
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

func parseSizeLimit(limitRaw *int) (*types.GuestUploadSizeLimit, error) {
	if limitRaw == nil {
		return nil, nil
	}
	// TODO: Check more rigorously
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload size limit must be a positive number")
	}

	limit := types.GuestUploadSizeLimit(*limitRaw)
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
