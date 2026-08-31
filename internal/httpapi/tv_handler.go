package httpapi

import (
	"net/http"
	"strconv"
)

func (s *server) listTVShows(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	items, err := s.library.ListTVShows(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list TV shows")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) getTVShow(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	result, err := s.library.TVShow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
