package httpapi

import (
	"net/http"

	"github.com/mediagrap/mediagrap/internal/auth"
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *server) setupStatus(w http.ResponseWriter, r *http.Request) {
	needs, err := s.auth.NeedsSetup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to check setup status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needsSetup": needs})
}

func (s *server) setup(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	session, err := s.auth.Setup(r.Context(), body.Username, body.Password)
	if err != nil {
		writeError(w, http.StatusConflict, "setup_unavailable", err.Error())
		return
	}
	s.setSession(w, r, session)
	writeJSON(w, http.StatusCreated, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	session, err := s.auth.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
		return
	}
	s.setSession(w, r, session)
	writeJSON(w, http.StatusOK, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	_ = s.auth.Logout(r.Context(), session.Token)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   s.runtime.SecureSessionCookie || r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) session(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, false)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}
