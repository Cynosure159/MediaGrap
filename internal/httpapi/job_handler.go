package httpapi

import (
	"net/http"
	"strconv"
)

func (s *server) listJobs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	if r.URL.Query().Get("activeScans") == "true" {
		after := int64(0)
		if value := r.URL.Query().Get("after"); value != "" {
			var err error
			after, err = strconv.ParseInt(value, 10, 64)
			if err != nil || after < 0 {
				writeError(w, http.StatusBadRequest, "invalid_cursor", "Invalid jobs cursor")
				return
			}
		}
		items, err := s.library.ActiveScans(r.Context(), after)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list active scans")
			return
		}
		next := int64(0)
		if len(items) == 100 {
			next = items[len(items)-1].ID
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": next})
		return
	}
	items, err := s.library.ListJobs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list jobs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) getJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_job", "Invalid job id")
		return
	}
	job, err := s.library.Job(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job_not_found", "Job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}
