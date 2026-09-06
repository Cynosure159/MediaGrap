package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/settings"
)

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, false)
	if !ok {
		return
	}
	view, err := s.settingsService.View(r.Context(), session.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) updateSettings(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	var update settings.Update
	if !decodeJSON(w, r, &update) {
		return
	}
	current, err := s.settingsService.Update(r.Context(), update, session.User.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	if err := s.metadata.ConfigureProviders(current.TMDbAPIKey, current.FanartTVAPIKey, current.FanartTVPersonalAPIKey, current.TMDbLanguage, current.FallbackLanguage, current.OutboundProxy, current.NoProxy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	s.logger.Info("Application settings updated",
		"tmdb_api_key_configured", current.TMDbAPIKey != "",
		"fanart_tv_api_key_configured", current.FanartTVAPIKey != "",
		"tmdb_language", current.TMDbLanguage,
		"fallback_language", current.FallbackLanguage,
		"outbound_proxy_configured", current.OutboundProxy != "",
		"no_proxy_configured", current.NoProxy != "",
	)
	view, err := s.settingsService.View(r.Context(), session.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load settings")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *server) testConnection(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	var body struct {
		Target string `json:"target"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Target != "tmdb" && body.Target != "fanart_tv" && body.Target != "proxy" {
		writeError(w, http.StatusBadRequest, "invalid_test_target", "Connection test target is invalid")
		return
	}
	var recent int
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM connection_test_results WHERE target=? AND tested_at>datetime('now','-10 seconds')`, body.Target).Scan(&recent); err == nil && recent > 0 {
		w.Header().Set("Retry-After", "10")
		writeError(w, http.StatusTooManyRequests, "connection_test_rate_limited", "Wait before testing this connection again")
		return
	}
	current, err := s.settingsService.Current(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load connection settings")
		return
	}
	configured := current.TMDbAPIKey != ""
	if body.Target == "fanart_tv" {
		configured = current.FanartTVAPIKey != ""
	}
	if body.Target == "proxy" {
		configured = current.OutboundProxy != "" && current.TMDbAPIKey != ""
	}
	if !configured {
		result := metadata.ConnectionTest{Target: body.Target, Status: "not_configured", Message: "Required credentials or proxy are not configured"}
		s.persistConnectionTest(r.Context(), result)
		writeJSON(w, http.StatusOK, result)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	result := s.metadata.TestConnection(ctx, body.Target)
	s.persistConnectionTest(r.Context(), result)
	s.logger.Info("connection test completed", "target", body.Target, "status", result.Status, "http_status", result.HTTPStatus, "duration_ms", result.DurationMS)
	writeJSON(w, http.StatusOK, result)
}

func (s *server) persistConnectionTest(ctx context.Context, result metadata.ConnectionTest) {
	_, _ = s.db.ExecContext(ctx, `INSERT INTO connection_test_results(target,status,http_status,duration_ms,message) VALUES(?,?,?,?,?)`, result.Target, result.Status, nullableStatus(result.HTTPStatus), result.DurationMS, result.Message)
}

func nullableStatus(status int) any {
	if status == 0 {
		return nil
	}
	return status
}
