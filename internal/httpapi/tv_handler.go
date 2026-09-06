package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
)

type tvArtworkPlanRequest struct {
	Scope        string                        `json:"scope"`
	SeasonNumber *int                          `json:"seasonNumber"`
	Selections   []metadata.TVArtworkSelection `json:"selections"`
}

func (s *server) listTVShows(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	items, err := s.library.ListTVShows(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list TV shows")
		return
	}
	for index := range items {
		if root, err := s.tvSourceRoot(r.Context(), items[index].SourceID); err == nil {
			showDir := filepath.Join(root, items[index].RelativePath)
			for _, name := range library.PosterFilenames {
				if info, err := os.Stat(filepath.Join(showDir, name)); err == nil && info.Mode().IsRegular() {
					items[index].PosterURL = "/api/v1/tv/shows/" + strconv.FormatInt(items[index].ID, 10) + "/poster"
					break
				}
			}
		}
		if items[index].PosterURL == "" {
			if record, err := s.metadata.TVRecord(r.Context(), items[index].ID); err == nil && record.PosterURL != "" {
				items[index].PosterURL = record.PosterURL
			}
		}
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
			metadata, err = s.metadata.SaveTV(r.Context(), fromNFO)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "Unable to cache local TV metadata")
				return
			}
			origin = "nfo"
		} else {
			origin = "empty"
		}
	}
	for _, asset := range result.Artwork {
		if asset.Kind == "poster" {
			metadata.PosterURL = tvArtworkURL(id, asset.ID)
		}
		if asset.Kind == "fanart" {
			metadata.BackdropURL = tvArtworkURL(id, asset.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"show": result.Show, "episodes": result.Episodes, "artwork": result.Artwork, "writable": result.Writable, "metadata": metadata, "metadataOrigin": origin})
}

func tvArtworkURL(showID int64, assetID string) string {
	return "/api/v1/tv/shows/" + strconv.FormatInt(showID, 10) + "/artwork/" + assetID
}

func (s *server) getTVArtworkCandidates(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	showID, scope, seasonNumber, ok := tvArtworkTargetScope(w, r)
	if !ok {
		return
	}
	items, err := s.metadata.CachedTVArtworkCandidates(r.Context(), showID, scope, seasonNumber)
	if err != nil {
		writeError(w, http.StatusBadRequest, "tv_artwork_candidates_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) scrapeTVArtworkCandidates(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	showID, scope, seasonNumber, ok := tvArtworkTargetScope(w, r)
	if !ok {
		return
	}
	if _, err := s.library.TVShow(r.Context(), showID); err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	items, err := s.metadata.TVArtworkCandidates(r.Context(), showID, scope, seasonNumber)
	if err != nil {
		s.logger.Warn("TV artwork candidates scrape failed", "show_id", showID, "scope", scope, "season_number", seasonNumber, "error", err)
		writeError(w, http.StatusBadRequest, "tv_artwork_candidates_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) getTVArtworkPreview(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	showID, _, _, ok := tvArtworkTargetScope(w, r)
	if !ok {
		return
	}
	preview, err := s.metadata.OpenTVArtworkPreview(r.Context(), showID, r.PathValue("candidate"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "tv_artwork_preview_failed", err.Error())
		return
	}
	defer preview.Body.Close()
	w.Header().Set("Content-Type", preview.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	if preview.ContentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(preview.ContentLength, 10))
	}
	_, _ = io.Copy(w, preview.Body)
}

func (s *server) previewTVArtworkPlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	showID, _, _, ok := tvArtworkTargetScope(w, r)
	if !ok {
		return
	}
	var body tvArtworkPlanRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Scope == "" {
		body.Scope = "show"
	}
	detail, err := s.library.TVShow(r.Context(), showID)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	plan, err := s.metadata.PreviewTVArtworkSelection(r.Context(), showID, body.Scope, body.SeasonNumber, body.Selections, filepath.Join(root, detail.Show.RelativePath), detail.Writable)
	if err != nil {
		writeError(w, http.StatusBadRequest, "tv_artwork_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func tvArtworkTargetScope(w http.ResponseWriter, r *http.Request) (int64, string, *int, bool) {
	showID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || showID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return 0, "", nil, false
	}
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = "show"
	}
	var seasonNumber *int
	if raw := strings.TrimSpace(r.URL.Query().Get("season")); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_season", "Invalid season number")
			return 0, "", nil, false
		}
		seasonNumber = &value
	}
	if scope == "show" && seasonNumber == nil {
		return showID, scope, nil, true
	}
	if scope == "season" && seasonNumber != nil && *seasonNumber >= 0 {
		return showID, scope, seasonNumber, true
	}
	writeError(w, http.StatusBadRequest, "invalid_artwork_scope", "Artwork scope must be show or a non-negative season")
	return 0, "", nil, false
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

func (s *server) scrapeTVSeason(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	showID, seasonNumber, detail, root, ok := s.tvScrapeTarget(w, r)
	if !ok {
		return
	}
	inputs := tvSeasonNFOInputs(root, detail, seasonNumber)
	if len(inputs) == 0 {
		writeError(w, http.StatusNotFound, "season_not_found", "TV season not found")
		return
	}
	record, err := s.metadata.ScrapeTVSeasonAndWrite(r.Context(), showID, seasonNumber, inputs, detail.Writable, s.library.Allowed)
	if err != nil {
		writeError(w, http.StatusBadRequest, "tv_season_scrape_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *server) scrapeTVEpisode(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	showID, seasonNumber, detail, root, ok := s.tvScrapeTarget(w, r)
	if !ok {
		return
	}
	episodeID, err := strconv.ParseInt(r.PathValue("episode"), 10, 64)
	if err != nil || episodeID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_episode", "Invalid TV episode id")
		return
	}
	for _, episode := range detail.Episodes {
		if episode.ID != episodeID || episode.SeasonNumber != seasonNumber {
			continue
		}
		input := tvEpisodeNFOInput(root, episode)
		record, scrapeErr := s.metadata.ScrapeTVEpisodeAndWrite(r.Context(), showID, seasonNumber, episode.EpisodeStart, input, detail.Writable, s.library.Allowed)
		if scrapeErr != nil {
			writeError(w, http.StatusBadRequest, "tv_episode_scrape_failed", scrapeErr.Error())
			return
		}
		writeJSON(w, http.StatusOK, record)
		return
	}
	writeError(w, http.StatusNotFound, "episode_not_found", "TV episode not found")
}

func (s *server) tvScrapeTarget(w http.ResponseWriter, r *http.Request) (int64, int, library.TVShowDetail, string, bool) {
	showID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || showID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return 0, 0, library.TVShowDetail{}, "", false
	}
	seasonNumber, err := strconv.Atoi(r.PathValue("season"))
	if err != nil || seasonNumber < 0 {
		writeError(w, http.StatusBadRequest, "invalid_season", "Invalid TV season number")
		return 0, 0, library.TVShowDetail{}, "", false
	}
	detail, err := s.library.TVShow(r.Context(), showID)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return 0, 0, library.TVShowDetail{}, "", false
	}
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return 0, 0, library.TVShowDetail{}, "", false
	}
	return showID, seasonNumber, detail, root, true
}

func (s *server) getTVShowPoster(w http.ResponseWriter, r *http.Request) {
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
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	showDir := filepath.Join(root, detail.Show.RelativePath)
	for _, name := range library.PosterFilenames {
		target := filepath.Join(showDir, name)
		if info, err := os.Stat(target); err == nil && info.Mode().IsRegular() {
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(target)))
			http.ServeFile(w, r, target)
			return
		}
	}
	meta, err := s.metadata.TVRecord(r.Context(), id)
	if err == nil && meta.PosterURL != "" {
		http.Redirect(w, r, meta.PosterURL, http.StatusFound)
		return
	}
	writeError(w, http.StatusNotFound, "poster_not_found", "TV show poster not found")
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

// getTVNFORaw returns only the selected show's, season's, or episode's local
// NFO. It deliberately has no filesystem path parameter.
func (s *server) getTVNFORaw(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	showID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || showID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_show", "Invalid TV show id")
		return
	}
	detail, err := s.library.TVShow(r.Context(), showID)
	if err != nil {
		writeError(w, http.StatusNotFound, "show_not_found", "TV show not found")
		return
	}
	root, err := s.tvSourceRoot(r.Context(), detail.Show.SourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_not_found", "TV source not found")
		return
	}
	kind := r.URL.Query().Get("kind")
	target := ""
	switch kind {
	case "show":
		target = filepath.Join(root, detail.Show.RelativePath, "tvshow.nfo")
	case "season":
		season, parseErr := strconv.Atoi(r.URL.Query().Get("season"))
		if parseErr != nil || season < 0 {
			writeError(w, http.StatusBadRequest, "invalid_season", "Invalid TV season number")
			return
		}
		for _, episode := range detail.Episodes {
			if episode.SeasonNumber == season {
				target = filepath.Join(filepath.Dir(filepath.Join(root, episode.RelativePath)), "season.nfo")
				break
			}
		}
	case "episode":
		episodeID, parseErr := strconv.ParseInt(r.URL.Query().Get("episode"), 10, 64)
		if parseErr != nil || episodeID < 1 {
			writeError(w, http.StatusBadRequest, "invalid_episode", "Invalid TV episode id")
			return
		}
		for _, episode := range detail.Episodes {
			if episode.ID == episodeID {
				absolute := filepath.Join(root, episode.RelativePath)
				target = strings.TrimSuffix(absolute, filepath.Ext(absolute)) + ".nfo"
				break
			}
		}
	default:
		writeError(w, http.StatusBadRequest, "invalid_nfo_kind", "Invalid TV NFO kind")
		return
	}
	if target == "" || !s.library.Allowed(target) {
		writeError(w, http.StatusNotFound, "nfo_not_found", "TV NFO not found")
		return
	}
	relativeTarget, relErr := filepath.Rel(root, target)
	if relErr != nil {
		writeError(w, http.StatusBadRequest, "nfo_read_failed", "Invalid TV NFO target")
		return
	}
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, http.StatusOK, map[string]any{"exists": false, "targetPath": relativeTarget, "content": ""})
		return
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		writeError(w, http.StatusBadRequest, "nfo_read_failed", "TV NFO is not a safe regular file")
		return
	}
	content, err := os.ReadFile(target)
	if err != nil {
		writeError(w, http.StatusNotFound, "nfo_not_found", "TV NFO not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"exists": true, "targetPath": relativeTarget, "content": string(content)})
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
	var firstID int64
	if len(detail.Episodes) > 0 {
		firstID = detail.Episodes[0].ID
	}
	inputs = append(inputs, metadata.TVNFOInput{MediaItemID: firstID, Kind: "show", TargetPath: filepath.Join(root, detail.Show.RelativePath, "tvshow.nfo")})
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

func tvSeasonNFOInputs(root string, detail library.TVShowDetail, seasonNumber int) []metadata.TVNFOInput {
	inputs := make([]metadata.TVNFOInput, 0)
	seenMedia := make(map[int64]struct{})
	for _, episode := range detail.Episodes {
		if episode.SeasonNumber != seasonNumber {
			continue
		}
		if len(inputs) == 0 {
			inputs = append(inputs, metadata.TVNFOInput{MediaItemID: episode.ID, Kind: "season", SeasonNumber: seasonNumber, TargetPath: filepath.Join(filepath.Dir(filepath.Join(root, episode.RelativePath)), "season.nfo")})
		}
		if _, exists := seenMedia[episode.ID]; exists {
			continue
		}
		seenMedia[episode.ID] = struct{}{}
		inputs = append(inputs, tvEpisodeNFOInput(root, episode))
	}
	return inputs
}

func tvEpisodeNFOInput(root string, episode library.TVEpisode) metadata.TVNFOInput {
	absolute := filepath.Join(root, episode.RelativePath)
	return metadata.TVNFOInput{MediaItemID: episode.ID, Kind: "episode", SeasonNumber: episode.SeasonNumber, EpisodeNumber: episode.EpisodeStart, TargetPath: strings.TrimSuffix(absolute, filepath.Ext(absolute)) + ".nfo"}
}
