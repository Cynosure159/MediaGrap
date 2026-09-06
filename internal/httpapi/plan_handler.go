package httpapi

import (
	"net/http"
)

func (s *server) getWritePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id := r.PathValue("id")
	plan, err := s.metadata.Plan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "plan_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) applyWritePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id := r.PathValue("id")
	plan, err := s.metadata.Apply(r.Context(), id, s.library.Allowed)
	if err != nil {
		writeError(w, http.StatusConflict, "write_conflict", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) getArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id := r.PathValue("id")
	plan, err := s.metadata.ArtworkPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "plan_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) applyArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id := r.PathValue("id")
	plan, err := s.metadata.QueueArtwork(r.Context(), id, s.library.Allowed)
	if err != nil {
		s.logger.Warn("Artwork apply failed", "plan_id", id, "error", err)
		writeError(w, http.StatusConflict, "artwork_write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) getTVArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	plan, err := s.metadata.TVArtworkPlan(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "plan_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *server) applyTVArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	plan, err := s.metadata.QueueTVArtwork(r.Context(), r.PathValue("id"), s.library.Allowed)
	if err != nil {
		writeError(w, http.StatusConflict, "tv_artwork_write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}
