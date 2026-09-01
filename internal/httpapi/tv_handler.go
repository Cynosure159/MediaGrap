package httpapi

import (
	"context"
	"encoding/base64"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
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
	metadata, err := s.metadata.TVRecord(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load TV metadata draft")
		return
	}
	origin := "draft"
	if metadata.Title == "" {
		root, rootErr := s.tvSourceRoot(r.Context(), result.Show.SourceID)
		if rootErr != nil {
			writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
			return
		}
		fromNFO, found, readErr := s.metadata.ReadExistingTVNFO(id, tvNFOInputs(root, result))
		if readErr != nil {
			writeError(w, http.StatusBadRequest, "nfo_read_failed", readErr.Error())
			return
		}
		if found {
			metadata, origin = fromNFO, "nfo"
		} else {
			origin = "empty"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"show": result.Show, "episodes": result.Episodes, "artwork": result.Artwork, "writable": result.Writable, "metadata": metadata, "metadataOrigin": origin})
}

func (s *server) getTVShowCandidates(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	show, err := s.library.TVShow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	year := show.Show.YearHint
	origin := "explicit"
	if query == "" {
		query, year, origin = show.Show.TitleHint, show.Show.YearHint, "show_directory"
	}
	s.logger.Info("TMDb TV search started", "show_id", id, "query_length", len([]rune(query)), "year", year, "query_origin", origin)
	candidates, err := s.metadata.SearchTV(r.Context(), query, year)
	if err != nil {
		s.logger.Warn("TMDb TV search failed", "show_id", id, "error", err)
		writeError(w, http.StatusBadRequest, "provider_unavailable", err.Error())
		return
	}
	s.logger.Info("TMDb TV search completed", "show_id", id, "candidate_count", len(candidates))
	writeJSON(w, http.StatusOK, map[string]any{"items": candidates})
}

func (s *server) selectTVShowCandidate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	show, err := s.library.TVShow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	var body metadataRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	root, err := s.tvSourceRoot(r.Context(), show.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	seasons := seasonsFromEpisodes(show.Episodes)
	record, err := s.metadata.SelectTVAndWrite(r.Context(), id, body.CandidateID, seasons, tvNFOInputs(root, show), show.Writable, s.library.Allowed)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provider_unavailable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *server) getTVArtwork(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	detail, err := s.library.TVShow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	assetID := r.PathValue("asset")
	var relative string
	for _, asset := range detail.Artwork {
		if asset.ID == assetID {
			relative = asset.RelativePath
			break
		}
	}
	if relative == "" {
		writeError(w, http.StatusNotFound, "artwork_not_found", "TV artwork not found")
		return
	}
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	decoded, decodeErr := base64.RawURLEncoding.DecodeString(assetID)
	if decodeErr != nil || string(decoded) != relative {
		writeError(w, http.StatusBadRequest, "invalid_artwork", "Invalid TV artwork")
		return
	}
	path := filepath.Join(root, relative)
	if !s.library.Allowed(path) {
		writeError(w, http.StatusForbidden, "invalid_artwork", "Invalid TV artwork")
		return
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "artwork_not_found", "TV artwork not found")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "artwork_not_found", "TV artwork not found")
		return
	}
	defer file.Close()
	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}

func (s *server) tvSourceRoot(ctx context.Context, sourceID int64) (string, error) {
	var root string
	err := s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, sourceID).Scan(&root)
	return root, err
}

func seasonsFromEpisodes(episodes []library.TVEpisode) []int {
	seen := make(map[int]struct{})
	seasons := make([]int, 0)
	for _, episode := range episodes {
		if _, ok := seen[episode.SeasonNumber]; ok {
			continue
		}
		seen[episode.SeasonNumber] = struct{}{}
		seasons = append(seasons, episode.SeasonNumber)
	}
	return seasons
}

func (s *server) previewTVNFOPlans(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	detail, err := s.library.TVShow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	metadata, err := s.metadata.TVRecord(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load TV metadata")
		return
	}
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	if len(detail.Episodes) == 0 {
		writeError(w, http.StatusBadRequest, "no_episodes", "No TV episodes are indexed")
		return
	}
	inputs := tvNFOInputs(root, detail)
	plans, err := s.metadata.PreviewTVNFO(r.Context(), metadata, inputs, detail.Writable)
	if err != nil {
		s.logger.Warn("TV NFO preview failed", "show_id", id, "error", err)
		writeError(w, http.StatusBadRequest, "write_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": plans})
}

func tvNFOInputs(root string, detail library.TVShowDetail) []metadata.TVNFOInput {
	inputs := make([]metadata.TVNFOInput, 0, len(detail.Episodes)*2+1)
	first := detail.Episodes[0]
	inputs = append(inputs, metadata.TVNFOInput{MediaItemID: first.ID, Kind: "show", TargetPath: filepath.Join(root, detail.Show.RelativePath, "tvshow.nfo")})
	seenSeasons := make(map[int]struct{})
	seenMedia := make(map[int64]struct{})
	for _, episode := range detail.Episodes {
		absolute := filepath.Join(root, episode.RelativePath)
		if _, exists := seenSeasons[episode.SeasonNumber]; !exists {
			seenSeasons[episode.SeasonNumber] = struct{}{}
			inputs = append(inputs, metadata.TVNFOInput{MediaItemID: episode.ID, Kind: "season", SeasonNumber: episode.SeasonNumber, TargetPath: filepath.Join(filepath.Dir(absolute), "season.nfo")})
		}
		if _, exists := seenMedia[episode.ID]; exists {
			continue
		}
		seenMedia[episode.ID] = struct{}{}
		inputs = append(inputs, metadata.TVNFOInput{MediaItemID: episode.ID, Kind: "episode", SeasonNumber: episode.SeasonNumber, EpisodeNumber: episode.EpisodeStart, TargetPath: strings.TrimSuffix(absolute, filepath.Ext(absolute)) + ".nfo"})
	}
	return inputs
}
