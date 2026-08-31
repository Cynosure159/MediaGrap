package httpapi

import (
	"net/http"
)

func (s *server) listJobs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	items, err := s.library.ListJobs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list jobs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
