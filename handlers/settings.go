package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (s Server) settingsPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := settingsFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// TODO: Do we need to lock?
		s.settings = settings

		if err := s.store.UpdateSettings(settings); err != nil {
			log.Printf("failed to save settings: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save settings: %v", err), http.StatusInternalServerError)
			return
		}

		//respondJSON(w, GuestLinkPostResponse{ID: string(gl.ID)})
	}
}

func settingsFromRequest(r *http.Request) (picoshare.Settings, error) {
	var payload struct {
		DefaultExpirationDays uint16 `json:"defaultExpirationDays"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("failed to decode JSON request: %v", err)
		return picoshare.Settings{}, err
	}

	// TODO: Actually parse this
	defaultExpirationDays := payload.DefaultExpirationDays

	return picoshare.Settings{
		DefaultEntryLifetime: time.Hour * 24 * time.Duration(defaultExpirationDays),
	}, nil
}
