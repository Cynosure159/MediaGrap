package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type renamePlanRequest struct {
	Pattern string `json:"pattern"`
}

type tvRenamePlanRequest struct {
	Pattern      string `json:"pattern"`
	SeasonNumber *int   `json:"seasonNumber,omitempty"`
	EpisodeID    *int64 `json:"episodeId,omitempty"`
}

func (s *server) previewMediaRenamePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}

	var body renamePlanRequest
	if !decodeJSON(w, r, &body) {
		return
	}

	plan, err := s.library.PreviewRenamePlan(r.Context(), id, body.Pattern)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rename_pattern", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (s *server) previewTVRenamePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}

	var body tvRenamePlanRequest
	if !decodeJSON(w, r, &body) {
		return
	}

	plan, err := s.library.PreviewTVRenamePlan(r.Context(), id, body.SeasonNumber, body.EpisodeID, body.Pattern)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_rename_pattern", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (s *server) getRenamePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_plan_id", "invalid plan id")
		return
	}
	
	plan, err := s.library.GetRenamePlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "plan not found")
		return
	}
	
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) applyRenamePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_plan_id", "invalid plan id")
		return
	}
	
	payload, _ := json.Marshal(map[string]string{"planId": id})
	job, err := s.library.QueuePayload(r.Context(), "rename_execute", nil, payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	
	writeJSON(w, http.StatusAccepted, job)
}
