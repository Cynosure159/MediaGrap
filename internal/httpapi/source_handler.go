package httpapi

import (
	"net/http"
	"strconv"
)

type sourceRequest struct {
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

func (s *server) listSources(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, false)
	if !ok {
		return
	}
	items, err := s.library.ListSources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list sources")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "user": session.User})
}

func (s *server) createSource(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	var body sourceRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	source, err := s.library.CreateSource(r.Context(), body.Name, body.RootPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_source", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, source)
}

func (s *server) scanSource(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireSession(w, r, true)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_source", "Invalid source id")
		return
	}
	job, err := s.library.QueueScan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
