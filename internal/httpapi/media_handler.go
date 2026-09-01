package httpapi

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mediagrap/mediagrap/internal/library"
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
	for index := range result.Items {
		if result.Items[index].Title == "" {
			location, locationErr := s.library.LocateMedia(r.Context(), result.Items[index].ID)
			if locationErr == nil {
				if hydrateErr := s.metadata.HydrateExistingNFO(r.Context(), result.Items[index].ID, location.AbsolutePath); hydrateErr == nil {
					if record, recordErr := s.metadata.Record(r.Context(), result.Items[index].ID); recordErr == nil {
						result.Items[index].Title = record.Title
						if record.PosterURL != "" {
							result.Items[index].PosterURL = record.PosterURL
						}
					}
				}
			}
		}
		if hasLocalArtwork(result.Items[index], "poster") {
			result.Items[index].PosterURL = mediaLocalArtworkURL(result.Items[index].ID, "poster")
		}
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
	if hasLocalArtwork(location.Item, "poster") {
		location.Item.PosterURL = mediaLocalArtworkURL(id, "poster")
	}
	record, err := s.metadata.Record(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load metadata")
		return
	}
	origin := "draft"
	if record.Title == "" {
		err = s.metadata.HydrateExistingNFO(r.Context(), id, location.AbsolutePath)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_nfo", err.Error())
			return
		}
		record, err = s.metadata.Record(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load metadata")
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

func mediaLocalArtworkURL(mediaID int64, kind string) string {
	return "/api/v1/media/" + strconv.FormatInt(mediaID, 10) + "/local-artwork/" + kind
}

func hasLocalArtwork(item library.MediaItem, kind string) bool {
	for _, asset := range item.Sidecars {
		name := strings.ToLower(strings.TrimSuffix(filepath.Base(asset.RelativePath), filepath.Ext(asset.RelativePath)))
		if asset.Kind != "image" {
			continue
		}
		if kind == "poster" && (name == "poster" || name == "folder" || name == "cover") {
			return true
		}
		if kind == "fanart" && (name == "fanart" || name == "backdrop" || name == "landscape") {
			return true
		}
	}
	return false
}

func (s *server) getMediaLocalArtwork(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_media", "Invalid media id")
		return
	}
	kind := r.PathValue("kind")
	if kind != "poster" && kind != "fanart" {
		writeError(w, http.StatusNotFound, "artwork_not_found", "Artwork not found")
		return
	}
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	var target string
	for _, asset := range location.Item.Sidecars {
		candidate := filepath.Join(filepath.Dir(location.AbsolutePath), filepath.Base(asset.RelativePath))
		candidateItem := location.Item
		candidateItem.Sidecars = []library.Sidecar{asset}
		if hasLocalArtwork(candidateItem, kind) {
			target = candidate
			break
		}
	}
	if target == "" || !s.library.Allowed(target) {
		writeError(w, http.StatusNotFound, "artwork_not_found", "Artwork not found")
		return
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "artwork_not_found", "Artwork not found")
		return
	}
	file, err := os.Open(target)
	if err != nil {
		writeError(w, http.StatusNotFound, "artwork_not_found", "Artwork not found")
		return
	}
	defer file.Close()
	if contentType := mime.TypeByExtension(filepath.Ext(target)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, filepath.Base(target), info.ModTime(), file)
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
	location, err := s.library.LocateMedia(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "media_not_found", "Media item not found")
		return
	}
	if !location.Writable {
		writeError(w, http.StatusBadRequest, "source_read_only", "The media source is read-only")
		return
	}
	record, err := s.metadata.SelectAndWrite(r.Context(), id, body.CandidateID, location.AbsolutePath, location.Writable, s.library.Allowed)
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
