package httpapi

import (
	"net/http"

	"github.com/mediagrap/mediagrap/internal/settings"
)

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	view, err := s.settingsService.View(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) updateSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
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
	if err := s.metadata.ConfigureFanart(current.FanartTVAPIKey, current.TMDbLanguage, current.OutboundProxy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	s.logger.Info("Application settings updated",
		"tmdb_api_key_configured", current.TMDbAPIKey != "",
		"fanart_tv_api_key_configured", current.FanartTVAPIKey != "",
		"tmdb_language", current.TMDbLanguage,
		"outbound_proxy_configured", current.OutboundProxy != "",
	)
	view, err := s.settingsService.View(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
		return
	}
	writeJSON(w, http.StatusOK, view)
}
