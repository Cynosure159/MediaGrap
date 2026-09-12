package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
)

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return false
	}
	return true
}

func (s *server) requireSession(w http.ResponseWriter, r *http.Request, csrf bool) (*auth.Session, bool) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue")
		return nil, false
	}
	session, err := s.auth.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue")
		return nil, false
	}
	if csrf && r.Header.Get("X-CSRF-Token") != session.CSRFToken {
		writeError(w, http.StatusForbidden, "csrf_failed", "Refresh the page and try again")
		return nil, false
	}
	return session, true
}

func (s *server) setSession(w http.ResponseWriter, r *http.Request, session *auth.Session) {
	secure := s.runtime.SecureSessionCookie || r.TLS != nil
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Expires:  session.ExpiresAt,
	})
}
