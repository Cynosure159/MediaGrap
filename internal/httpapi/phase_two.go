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

func (s *server) mediaDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, r.Method == http.MethodPost); !ok {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/media/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, 400, "invalid_media", "Invalid media id")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, 404, "media_not_found", "Media item not found")
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		record, err := s.metadata.Record(r.Context(), id)
		if err != nil {
			writeError(w, 500, "internal_error", "Unable to load metadata")
			return
		}
		origin := "draft"
		if record.Title == "" {
			record, _, err = s.metadata.ReadExistingNFO(location.AbsolutePath, id)
			if err != nil {
				writeError(w, 422, "invalid_nfo", err.Error())
				return
			}
			if record.Title != "" {
				origin = "nfo"
			} else {
				origin = "empty"
			}
		}
		writeJSON(w, 200, map[string]any{"item": location.Item, "metadata": record, "metadataOrigin": origin, "writable": location.Writable})
		return
	}
	if len(parts) != 2 {
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	switch parts[1] {
	case "candidates":
		if r.Method != http.MethodGet {
			break
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
			writeError(w, 400, "provider_unavailable", err.Error())
			return
		}
		s.logger.Info("TMDb movie search completed", "media_item_id", id, "candidate_count", len(candidates))
		writeJSON(w, 200, map[string]any{"items": candidates})
		return
	case "select":
		if r.Method != http.MethodPost {
			break
		}
		var body metadataRequest
		if !decodeJSON(w, r, &body) {
			return
		}
		record, err := s.metadata.Select(r.Context(), id, body.CandidateID)
		if err != nil {
			writeError(w, 400, "provider_unavailable", err.Error())
			return
		}
		writeJSON(w, 200, record)
		return
	case "metadata":
		if r.Method != http.MethodPost {
			break
		}
		var body metadataRequest
		if !decodeJSON(w, r, &body) {
			return
		}
		body.MediaItemID = id
		record, err := s.metadata.Save(r.Context(), body.Record)
		if err != nil {
			writeError(w, 400, "invalid_metadata", err.Error())
			return
		}
		writeJSON(w, 200, record)
		return
	case "write-plans":
		if r.Method != http.MethodPost {
			break
		}
		var body metadataRequest
		if !decodeJSON(w, r, &body) {
			return
		}
		body.MediaItemID = id
		plan, err := s.metadata.Preview(r.Context(), body.Record, location.AbsolutePath, location.Writable)
		if err != nil {
			writeError(w, 400, "write_preview_failed", err.Error())
			return
		}
		writeJSON(w, 201, plan)
		return
	default:
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	writeError(w, 405, "method_not_allowed", "Method not allowed")
}

func (s *server) writePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, r.Method == http.MethodPost); !ok {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/write-plans/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	if r.Method == http.MethodGet && len(parts) == 1 {
		plan, err := s.metadata.Plan(r.Context(), parts[0])
		if err != nil {
			writeError(w, 404, "plan_not_found", err.Error())
			return
		}
		writeJSON(w, 200, plan)
		return
	}
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "apply" {
		plan, err := s.metadata.Apply(r.Context(), parts[0], s.library.Allowed)
		if err != nil {
			writeError(w, 409, "write_conflict", err.Error())
			return
		}
		writeJSON(w, 200, plan)
		return
	}
	writeError(w, 404, "not_found", "Endpoint not found")
}
