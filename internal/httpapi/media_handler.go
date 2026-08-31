package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mediagrap/mediagrap/internal/metadata"
)

type metadataRequest struct {
	metadata.Record
	CandidateID string `json:"candidateId"`
}

func (s *server) listMedia(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	result, err := s.library.ListMedia(r.Context(), r.URL.Query().Get("q"), page, size)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list media")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) getMedia(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	record, err := s.metadata.Record(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load metadata")
		return
	}
	origin := "draft"
	if record.Title == "" {
		record, _, err = s.metadata.ReadExistingNFO(location.AbsolutePath, id)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_nfo", err.Error())
			return
		}
		if record.Title != "" {
			origin = "nfo"
		} else {
			origin = "empty"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"item":           location.Item,
		"metadata":       record,
		"metadataOrigin": origin,
		"writable":       location.Writable,
	})
}

func (s *server) getMediaCandidates(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	year := location.Item.YearHint
	origin := "explicit"
	if query == "" {
		query, year, origin = s.library.MetadataSearchHint(location.Item)
	}
	s.logger.Info("TMDb movie search started", "media_item_id", id, "query_length", len([]rune(query)), "year", year, "query_origin", origin)
	candidates, err := s.metadata.Search(r.Context(), query, year)
	if err != nil {
		s.logger.Warn("TMDb movie search failed", "media_item_id", id, "error", err)
		writeError(w, http.StatusBadRequest, "provider_unavailable", err.Error())
		return
	}
	s.logger.Info("TMDb movie search completed", "media_item_id", id, "candidate_count", len(candidates))
	writeJSON(w, http.StatusOK, map[string]any{"items": candidates})
}

func (s *server) selectMediaCandidate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	var body metadataRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	record, err := s.metadata.Select(r.Context(), id, body.CandidateID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provider_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *server) saveMediaMetadata(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	var body metadataRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	body.MediaItemID = id
	record, err := s.metadata.Save(r.Context(), body.Record)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_metadata", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *server) previewMediaWritePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	var body metadataRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	body.MediaItemID = id
	plan, err := s.metadata.Preview(r.Context(), body.Record, location.AbsolutePath, location.Writable)
	if err != nil {
		writeError(w, http.StatusBadRequest, "write_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func (s *server) previewMediaArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	var body metadataRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	body.MediaItemID = id
	plan, err := s.metadata.PreviewArtwork(r.Context(), body.Record, location.AbsolutePath, location.Writable)
	if err != nil {
		s.logger.Warn("Artwork preview failed", "media_item_id", id, "error", err)
		writeError(w, http.StatusBadRequest, "artwork_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}
