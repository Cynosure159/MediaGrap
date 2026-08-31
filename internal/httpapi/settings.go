package httpapi

import (
	"net/http"

	"github.com/mediagrap/mediagrap/internal/settings"
)

func (s *server) settings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, r.Method == http.MethodPut); !ok {
		return
	}
	if r.Method == http.MethodGet {
		view, err := s.settingsService.View(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
			return
		}
		writeJSON(w, http.StatusOK, view)
		return
	}
	var update settings.Update
	if !decodeJSON(w, r, &update) {
		return
	}
	current, err := s.settingsService.Update(r.Context(), update)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	if err := s.metadata.ConfigureTMDb(current.TMDbAPIKey, current.TMDbLanguage, current.OutboundProxy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	s.logger.Info("Application settings updated", "tmdb_api_key_configured", current.TMDbAPIKey != "", "tmdb_language", current.TMDbLanguage, "outbound_proxy_configured", current.OutboundProxy != "")
	view, err := s.settingsService.View(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
		return
	}
	writeJSON(w, http.StatusOK, view)
}
