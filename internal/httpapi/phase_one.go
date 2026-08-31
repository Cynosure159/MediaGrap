package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type sourceRequest struct {
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

func (s *server) setupStatus(w http.ResponseWriter, r *http.Request) {
	needs, err := s.auth.NeedsSetup(r.Context())
	if err != nil {
		writeError(w, 500, "internal_error", "Unable to check setup status")
		return
	}
	writeJSON(w, 200, map[string]bool{"needsSetup": needs})
}
func (s *server) setup(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	session, err := s.auth.Setup(r.Context(), body.Username, body.Password)
	if err != nil {
		writeError(w, 409, "setup_unavailable", err.Error())
		return
	}
	setSession(w, r, session)
	writeJSON(w, 201, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	session, err := s.auth.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		writeError(w, 401, "invalid_credentials", "Invalid username or password")
		return
	}
	setSession(w, r, session)
	writeJSON(w, 200, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	_ = s.auth.Logout(r.Context(), session.Token)
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}
func (s *server) session(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, false)
	if !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"user": session.User, "csrfToken": session.CSRFToken})
}
func (s *server) sources(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, r.Method == http.MethodPost)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		var body sourceRequest
		if !decodeJSON(w, r, &body) {
			return
		}
		source, err := s.library.CreateSource(r.Context(), body.Name, body.RootPath)
		if err != nil {
			writeError(w, 400, "invalid_source", err.Error())
			return
		}
		writeJSON(w, 201, source)
		return
	}
	items, err := s.library.ListSources(r.Context())
	if err != nil {
		writeError(w, 500, "internal_error", "Unable to list sources")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "user": session.User})
}
func (s *server) scanSource(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sources/"), "/")
	if len(parts) != 2 || parts[1] != "scans" {
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, 400, "invalid_source", "Invalid source id")
		return
	}
	job, err := s.library.QueueScan(r.Context(), id)
	if err != nil {
		writeError(w, 404, "source_not_found", err.Error())
		return
	}
	writeJSON(w, 202, job)
}
func (s *server) media(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	result, err := s.library.ListMedia(r.Context(), r.URL.Query().Get("q"), page, size)
	if err != nil {
		writeError(w, 500, "internal_error", "Unable to list media")
		return
	}
	writeJSON(w, 200, result)
}
func (s *server) jobs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	items, err := s.library.ListJobs(r.Context())
	if err != nil {
		writeError(w, 500, "internal_error", "Unable to list jobs")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *server) requireSession(w http.ResponseWriter, r *http.Request, csrf bool) (*auth.Session, bool) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err != nil {
		writeError(w, 401, "unauthorized", "Sign in to continue")
		return nil, false
	}
	session, err := s.auth.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, 401, "unauthorized", "Sign in to continue")
		return nil, false
	}
	if csrf && r.Header.Get("X-CSRF-Token") != session.CSRFToken {
		writeError(w, 403, "csrf_failed", "Refresh the page and try again")
		return nil, false
	}
	return session, true
}
func setSession(w http.ResponseWriter, r *http.Request, session *auth.Session) {
	secure := r.TLS != nil
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: session.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure, Expires: session.ExpiresAt})
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, 400, "invalid_json", "Request body must be valid JSON")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}, "timestamp": time.Now().UTC().Format(time.RFC3339)})
}
