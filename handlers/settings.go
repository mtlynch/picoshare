package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (s Server) settingsPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := settingsFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		s.settings.Update(settings)

		if err := s.getDB(r).UpdateSettings(settings); err != nil {
			log.Printf("failed to save settings: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save settings: %v", err), http.StatusInternalServerError)
			return
		}
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

	defaultExpiration, err := parse.FileLifetime(payload.DefaultExpirationDays)
	if err != nil {
		return picoshare.Settings{}, err
	}

	return picoshare.Settings{
		DefaultFileLifetime: defaultExpiration,
	}, nil
}
