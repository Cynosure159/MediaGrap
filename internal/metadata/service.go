package metadata

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/artwork"
	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/jobs"
	"github.com/mediagrap/mediagrap/internal/nfo"
	"github.com/mediagrap/mediagrap/internal/providers/fanart"
)

type Record struct {
	MediaItemID    int64    `json:"mediaItemId"`
	Provider       string   `json:"provider"`
	ProviderID     string   `json:"providerId"`
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"originalTitle"`
	Year           *int     `json:"year"`
	Overview       string   `json:"overview"`
	RuntimeMinutes *int     `json:"runtimeMinutes"`
	Genres         []string `json:"genres"`
	PosterURL      string   `json:"posterUrl"`
	BackdropURL    string   `json:"backdropUrl"`
	Rating         *float64 `json:"rating"`
	Votes          *int     `json:"votes"`
	ContentRating  string   `json:"contentRating"`
	Directors      []string `json:"directors"`
	Writers        []string `json:"writers"`
	Studios        []string `json:"studios"`
	Cast           []Person `json:"cast"`
	LockedFields   []string `json:"lockedFields"`
	UpdatedAt      string   `json:"updatedAt"`
}
type WritePlan struct {
	ID          string `json:"id"`
	MediaItemID int64  `json:"mediaItemId"`
	TargetPath  string `json:"targetPath"`
	Content     string `json:"content"`
	State       string `json:"state"`
	CreatedAt   string `json:"createdAt"`
	Conflict    bool   `json:"conflict"`
	WillReplace bool   `json:"willReplace"`
}
type ArtworkAsset struct {
	Kind            string `json:"kind"`
	CandidateID     string `json:"candidateId,omitempty"`
	Provider        string `json:"provider,omitempty"`
	ProviderAssetID string `json:"providerAssetId,omitempty"`
	SourceURL       string `json:"sourceUrl"`
	PreviewURL      string `json:"previewUrl,omitempty"`
	Language        string `json:"language,omitempty"`
	Likes           int    `json:"likes,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	MimeType        string `json:"mimeType,omitempty"`
	TargetPath      string `json:"targetPath"`
	Conflict        bool   `json:"conflict"`
	WillReplace     bool   `json:"willReplace"`
}
type ArtworkPlan struct {
	ID          string         `json:"id"`
	MediaItemID int64          `json:"mediaItemId"`
	State       string         `json:"state"`
	CreatedAt   string         `json:"createdAt"`
	Assets      []ArtworkAsset `json:"assets"`
}
type ArtworkCandidate struct {
	ID              string `json:"id"`
	MediaItemID     int64  `json:"mediaItemId"`
	Provider        string `json:"provider"`
	ProviderAssetID string `json:"providerAssetId"`
	Kind            string `json:"kind"`
	SourceURL       string `json:"sourceUrl"`
	PreviewURL      string `json:"previewUrl"`
	Language        string `json:"language"`
	Likes           int    `json:"likes"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	MimeType        string `json:"mimeType"`
}
type ArtworkSelection struct {
	Kind        string `json:"kind"`
	CandidateID string `json:"candidateId"`
}

type ArtworkPreview struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}
type TVRecord struct {
	ShowID        int64              `json:"showId"`
	Provider      string             `json:"provider"`
	ProviderID    string             `json:"providerId"`
	TVDBID        string             `json:"tvdbId"`
	IMDbID        string             `json:"imdbId"`
	Title         string             `json:"title"`
	OriginalTitle string             `json:"originalTitle"`
	Year          *int               `json:"year"`
	Overview      string             `json:"overview"`
	Genres        []string           `json:"genres"`
	PosterURL     string             `json:"posterUrl"`
	BackdropURL   string             `json:"backdropUrl"`
	Rating        *float64           `json:"rating"`
	Votes         *int               `json:"votes"`
	Status        string             `json:"status"`
	Network       string             `json:"network"`
	Cast          []Person           `json:"cast"`
	Episodes      []TVEpisodeDetails `json:"episodes"`
	UpdatedAt     string             `json:"updatedAt"`
}
type TVNFOInput struct {
	MediaItemID   int64
	Kind          string
	TargetPath    string
	SeasonNumber  int
	EpisodeNumber int
}
type Service struct {
	db            *sql.DB
	repo          Repository
	provider      Provider
	artworkMu     sync.RWMutex
	artworkClient *http.Client
	artworkLocks  sync.Map
	fanart        *fanart.Client
	jobQueue      artworkJobQueue
	logger        *slog.Logger
}

type ConnectionTest struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	DurationMS int64  `json:"durationMs"`
	Message    string `json:"message"`
}

type artworkJobQueue interface {
	QueuePayload(context.Context, string, *int64, []byte) (jobs.Job, error)
	RegisterJobHandler(string, func(context.Context, jobs.Job, func(int, string)) error)
}

func NewService(db *sql.DB, provider Provider) *Service {
	service := &Service{db: db, repo: NewRepository(db), provider: provider, artworkClient: http.DefaultClient, logger: slog.Default()}
	if tmdb, ok := provider.(*TMDb); ok {
		service.artworkClient = tmdb.HTTPClient()
		service.logger = tmdb.Logger()
	}
	return service
}

func (s *Service) ConfigureTMDb(apiKey, language, proxy string) error {
	provider, ok := s.provider.(*TMDb)
	if !ok {
		return errors.New("TMDb provider is unavailable")
	}
	client, err := NewOutboundClient(proxy)
	if err != nil {
		return err
	}
	provider.Configure(client, apiKey, language)
	s.artworkMu.Lock()
	s.artworkClient = client
	s.artworkMu.Unlock()
	return nil
}

func (s *Service) SetFanartProvider(provider *fanart.Client) {
	s.fanart = provider
}

func (s *Service) ConfigureFanart(apiKey, language, proxy string) error {
	if s.fanart == nil {
		if strings.TrimSpace(apiKey) == "" {
			return nil
		}
		return errors.New("Fanart.tv provider is unavailable")
	}
	client, err := NewOutboundClient(proxy)
	if err != nil {
		return err
	}
	s.fanart.Configure(client, apiKey, language)
	s.artworkMu.Lock()
	s.artworkClient = client
	s.artworkMu.Unlock()
	return nil
}

func (s *Service) ConfigureProviders(apiKey, fanartAPIKey, fanartPersonalAPIKey, language, fallbackLanguage, proxy, noProxy string) error {
	client, err := NewOutboundClient(proxy, noProxy)
	if err != nil {
		return err
	}
	provider, ok := s.provider.(*TMDb)
	if !ok {
		return errors.New("TMDb provider is unavailable")
	}
	provider.ConfigureAdvanced(client, apiKey, language, fallbackLanguage)
	s.artworkMu.Lock()
	s.artworkClient = client
	s.artworkMu.Unlock()
	if s.fanart != nil {
		s.fanart.ConfigureKeys(client, fanartAPIKey, fanartPersonalAPIKey, language)
	} else if strings.TrimSpace(fanartAPIKey) != "" || strings.TrimSpace(fanartPersonalAPIKey) != "" {
		return errors.New("Fanart.tv provider is unavailable")
	}
	return nil
}

func (s *Service) TestConnection(ctx context.Context, target string) ConnectionTest {
	started := time.Now()
	result := ConnectionTest{Target: target, Status: "failed"}
	var err error
	switch target {
	case "tmdb", "proxy":
		provider, ok := s.provider.(*TMDb)
		if !ok {
			err = errors.New("TMDb provider is unavailable")
		} else {
			result.HTTPStatus, err = provider.Ping(ctx)
		}
	case "fanart_tv":
		if s.fanart == nil {
			err = errors.New("Fanart.tv provider is unavailable")
		} else {
			result.HTTPStatus, err = s.fanart.Ping(ctx)
		}
	default:
		err = errors.New("unsupported connection test target")
	}
	result.DurationMS = time.Since(started).Milliseconds()
	if err == nil {
		result.Status, result.Message = "reachable", "Connection succeeded"
	} else {
		result.Message = err.Error()
	}
	return result
}

func (s *Service) SetJobService(queue artworkJobQueue) {
	s.jobQueue = queue
	if queue != nil {
		queue.RegisterJobHandler("artwork_download", s.handleArtworkDownloadJob)
		queue.RegisterJobHandler("tv_artwork_download", s.handleTVArtworkDownloadJob)
	}
}

func (s *Service) Search(ctx context.Context, query string, year *int) ([]Candidate, error) {
	return s.provider.SearchMovies(ctx, strings.TrimSpace(query), year)
}
func (s *Service) Select(ctx context.Context, itemID int64, providerID string) (Record, error) {
	s.logger.Info("TMDb movie selection started", "media_item_id", itemID, "provider_id", providerID)
	details, err := s.provider.Movie(ctx, providerID)
	if err != nil {
		s.logger.Warn("TMDb movie selection failed", "media_item_id", itemID, "provider_id", providerID, "error", err)
		return Record{}, err
	}
	record := Record{MediaItemID: itemID, Provider: "tmdb", ProviderID: details.ID, Title: details.Title, OriginalTitle: details.OriginalTitle, Year: details.Year, Overview: details.Overview, RuntimeMinutes: details.Runtime, Genres: details.Genres, PosterURL: details.PosterURL, BackdropURL: details.BackdropURL, Rating: details.Rating, Votes: details.Votes, ContentRating: details.ContentRating, Directors: details.Directors, Writers: details.Writers, Studios: details.Studios, Cast: details.Cast}
	saved, err := s.Save(ctx, record)
	if err != nil {
		s.logger.Warn("TMDb movie metadata save failed", "media_item_id", itemID, "error", err)
		return Record{}, err
	}
	s.logger.Info("TMDb movie selection completed", "media_item_id", itemID, "cast_count", len(saved.Cast), "director_count", len(saved.Directors), "has_rating", saved.Rating != nil)
	return saved, nil
}

// SelectAndWrite replaces an item's metadata with the explicitly selected
// provider result and immediately writes its Kodi NFO using the same atomic
// write and audit path as an applied write plan.
func (s *Service) SelectAndWrite(ctx context.Context, itemID int64, providerID, mediaPath string, writable bool, allowed func(string) bool) (Record, error) {
	record, err := s.Select(ctx, itemID, providerID)
	if err != nil {
		return Record{}, err
	}
	plan, err := s.Preview(ctx, record, mediaPath, writable)
	if err != nil {
		return Record{}, err
	}
	if _, err = s.Apply(ctx, plan.ID, allowed); err != nil {
		return Record{}, err
	}
	s.logger.Info("TMDb movie selection NFO write completed", "media_item_id", itemID, "provider_id", providerID)
	return record, nil
}

func (s *Service) SearchTV(ctx context.Context, query string, year *int) ([]Candidate, error) {
	provider, ok := s.provider.(TVProvider)
	if !ok {
		return nil, errors.New("TV metadata provider is unavailable")
	}
	return provider.SearchTV(ctx, strings.TrimSpace(query), year)
}

func (s *Service) SelectTV(ctx context.Context, showID int64, providerID string, seasons []int) (TVRecord, error) {
	provider, ok := s.provider.(TVProvider)
	if !ok {
		return TVRecord{}, errors.New("TV metadata provider is unavailable")
	}
	s.logger.Info("TMDb TV metadata replacement started", "show_id", showID, "provider_id", providerID, "season_count", len(seasons))
	details, err := provider.TV(ctx, providerID)
	if err != nil {
		s.logger.Warn("TMDb TV metadata replacement failed", "show_id", showID, "provider_id", providerID, "error", err)
		return TVRecord{}, err
	}
	existing, _ := s.TVRecord(ctx, showID)
	if details.TVDBID == "" {
		details.TVDBID = existing.TVDBID
	}
	if details.IMDbID == "" {
		details.IMDbID = existing.IMDbID
	}
	episodes := make([]TVEpisodeDetails, 0)
	for _, season := range seasons {
		remote, fetchErr := provider.TVSeason(ctx, providerID, season)
		if fetchErr != nil {
			s.logger.Warn("TMDb TV season lookup failed", "show_id", showID, "season_number", season, "error", fetchErr)
			return TVRecord{}, fetchErr
		}
		episodes = append(episodes, remote...)
	}
	record := TVRecord{ShowID: showID, Provider: "tmdb", ProviderID: details.ID, TVDBID: details.TVDBID, IMDbID: details.IMDbID, Title: details.Title, OriginalTitle: details.OriginalTitle, Year: details.Year, Overview: details.Overview, Genres: details.Genres, PosterURL: details.PosterURL, BackdropURL: details.BackdropURL, Rating: details.Rating, Votes: details.Votes, Status: details.Status, Network: details.Network, Cast: details.Cast, Episodes: episodes}
	saved, err := s.SaveTV(ctx, record)
	if err != nil {
		s.logger.Warn("TMDb TV metadata replacement save failed", "show_id", showID, "error", err)
		return TVRecord{}, err
	}
	s.logger.Info("TMDb TV metadata replacement completed", "show_id", showID, "episode_count", len(saved.Episodes), "cast_count", len(saved.Cast), "has_rating", saved.Rating != nil)
	return saved, nil
}

// SelectTVAndWrite replaces a show's selected metadata and immediately applies
// one atomic Kodi NFO write for every supplied show, season, and episode target.
// Image files are deliberately outside this operation.
func (s *Service) SelectTVAndWrite(ctx context.Context, showID int64, providerID string, seasons []int, inputs []TVNFOInput, writable bool, allowed func(string) bool) (TVRecord, error) {
	record, err := s.SelectTV(ctx, showID, providerID, seasons)
	if err != nil {
		return TVRecord{}, err
	}
	plans, err := s.PreviewTVNFO(ctx, record, inputs, writable)
	if err != nil {
		return TVRecord{}, err
	}
	for _, plan := range plans {
		if _, err := s.Apply(ctx, plan.ID, allowed); err != nil {
			return TVRecord{}, err
		}
	}
	s.logger.Info("TMDb TV selection NFO writes completed", "show_id", showID, "provider_id", providerID, "plan_count", len(plans))
	return record, nil
}

// ScrapeTVSeasonAndWrite refreshes only one indexed season. It retains show and
// other-season metadata, then writes only that season's NFO and episode NFOs.
func (s *Service) ScrapeTVSeasonAndWrite(ctx context.Context, showID int64, seasonNumber int, inputs []TVNFOInput, writable bool, allowed func(string) bool) (TVRecord, error) {
	provider, ok := s.provider.(TVProvider)
	if !ok {
		return TVRecord{}, errors.New("TV metadata provider is unavailable")
	}
	record, err := s.TVRecord(ctx, showID)
	if err != nil {
		return TVRecord{}, err
	}
	if record.Title == "" || record.ProviderID == "" {
		return TVRecord{}, errors.New("match the TV show before scraping an individual season")
	}
	s.logger.Info("TMDb TV season scrape started", "show_id", showID, "season_number", seasonNumber)
	remote, err := provider.TVSeason(ctx, record.ProviderID, seasonNumber)
	if err != nil {
		return TVRecord{}, err
	}
	record.Episodes = mergeTVEpisodes(record.Episodes, remote, func(item TVEpisodeDetails) bool { return item.SeasonNumber == seasonNumber })
	saved, err := s.SaveTV(ctx, record)
	if err != nil {
		return TVRecord{}, err
	}
	plans, err := s.PreviewTVNFO(ctx, saved, inputs, writable)
	if err != nil {
		return TVRecord{}, err
	}
	for _, plan := range plans {
		if _, err := s.Apply(ctx, plan.ID, allowed); err != nil {
			return TVRecord{}, err
		}
	}
	s.logger.Info("TMDb TV season scrape completed", "show_id", showID, "season_number", seasonNumber, "plan_count", len(plans))
	return saved, nil
}

// ScrapeTVEpisodeAndWrite refreshes a single indexed episode without changing
// the rest of the show. The owning show must already be matched to TMDb.
func (s *Service) ScrapeTVEpisodeAndWrite(ctx context.Context, showID int64, seasonNumber int, episodeNumber int, input TVNFOInput, writable bool, allowed func(string) bool) (TVRecord, error) {
	provider, ok := s.provider.(TVProvider)
	if !ok {
		return TVRecord{}, errors.New("TV metadata provider is unavailable")
	}
	record, err := s.TVRecord(ctx, showID)
	if err != nil {
		return TVRecord{}, err
	}
	if record.Title == "" || record.ProviderID == "" {
		return TVRecord{}, errors.New("match the TV show before scraping an individual episode")
	}
	remote, err := provider.TVSeason(ctx, record.ProviderID, seasonNumber)
	if err != nil {
		return TVRecord{}, err
	}
	var selected *TVEpisodeDetails
	for index := range remote {
		if remote[index].SeasonNumber == seasonNumber && remote[index].EpisodeNumber == episodeNumber {
			selected = &remote[index]
			break
		}
	}
	if selected == nil {
		return TVRecord{}, fmt.Errorf("TMDb episode S%02dE%02d is unavailable", seasonNumber, episodeNumber)
	}
	record.Episodes = mergeTVEpisodes(record.Episodes, []TVEpisodeDetails{*selected}, func(item TVEpisodeDetails) bool {
		return item.SeasonNumber == seasonNumber && item.EpisodeNumber == episodeNumber
	})
	saved, err := s.SaveTV(ctx, record)
	if err != nil {
		return TVRecord{}, err
	}
	plans, err := s.PreviewTVNFO(ctx, saved, []TVNFOInput{input}, writable)
	if err != nil {
		return TVRecord{}, err
	}
	if _, err := s.Apply(ctx, plans[0].ID, allowed); err != nil {
		return TVRecord{}, err
	}
	s.logger.Info("TMDb TV episode scrape completed", "show_id", showID, "season_number", seasonNumber, "episode_number", episodeNumber)
	return saved, nil
}

func mergeTVEpisodes(existing, replacement []TVEpisodeDetails, remove func(TVEpisodeDetails) bool) []TVEpisodeDetails {
	merged := make([]TVEpisodeDetails, 0, len(existing)+len(replacement))
	for _, item := range existing {
		if !remove(item) {
			merged = append(merged, item)
		}
	}
	return append(merged, replacement...)
}

func (s *Service) TVRecord(ctx context.Context, showID int64) (TVRecord, error) {
	return s.repo.GetTVRecord(ctx, showID)
}

// ReadExistingTVNFO reads existing Kodi show and episode NFOs without
// persisting or changing them. It is used only when no selected draft exists.
func (s *Service) ReadExistingTVNFO(showID int64, inputs []TVNFOInput) (TVRecord, bool, error) {
	record := normalizedTV(TVRecord{ShowID: showID, Provider: "nfo"})
	found := false
	for _, input := range inputs {
		contents, exists, err := readRegularNFO(input.TargetPath)
		if err != nil {
			return TVRecord{}, false, err
		}
		if !exists {
			continue
		}
		switch input.Kind {
		case "show":
			show, err := nfo.ParseTVShow(contents)
			if err != nil {
				return TVRecord{}, false, fmt.Errorf("parse existing TV show NFO: %w", err)
			}
			if show.Title == "" {
				return TVRecord{}, false, errors.New("existing TV show NFO has no title")
			}
			record.Title, record.OriginalTitle, record.Year, record.Overview = show.Title, show.OriginalTitle, show.Year, show.Plot
			record.ProviderID, record.TVDBID, record.Genres, record.PosterURL, record.BackdropURL = show.TMDbID, show.TVDBID, show.Genres, show.PosterURL, show.BackdropURL
			record.Rating, record.Votes, record.Status, record.Network, record.Cast = show.Rating, show.Votes, show.Status, show.Network, metadataPeople(show.Cast)
			found = true
		case "episode":
			episode, err := nfo.ParseEpisode(contents)
			if err != nil {
				return TVRecord{}, false, fmt.Errorf("parse existing episode NFO: %w", err)
			}
			record.Episodes = append(record.Episodes, TVEpisodeDetails{SeasonNumber: episode.Season, EpisodeNumber: episode.Episode, Title: episode.Title, Overview: episode.Plot, AirDate: episode.AirDate, RuntimeMinutes: episode.Runtime, StillURL: episode.StillURL})
			found = true
		}
	}
	return record, found, nil
}

func readRegularNFO(path string) ([]byte, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("inspect existing NFO: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, false, errors.New("existing NFO must be a regular file")
	}
	if info.Size() > 1<<20 {
		return nil, false, errors.New("existing NFO exceeds the 1 MiB safety limit")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read existing NFO: %w", err)
	}
	return contents, true, nil
}

func (s *Service) SaveTV(ctx context.Context, record TVRecord) (TVRecord, error) {
	if record.ShowID < 1 || strings.TrimSpace(record.Title) == "" {
		return TVRecord{}, errors.New("TV show title is required")
	}
	if err := s.repo.SaveTVRecord(ctx, record); err != nil {
		return TVRecord{}, err
	}
	return s.TVRecord(ctx, record.ShowID)
}

func normalizedTV(record TVRecord) TVRecord {
	if record.Genres == nil {
		record.Genres = []string{}
	}
	if record.Cast == nil {
		record.Cast = []Person{}
	}
	if record.Episodes == nil {
		record.Episodes = []TVEpisodeDetails{}
	}
	return record
}

// PreviewTVNFO creates reviewable, independent write plans for a selected TV
// show. It does not write any media-side file; callers must apply each plan.
func (s *Service) PreviewTVNFO(ctx context.Context, record TVRecord, inputs []TVNFOInput, writable bool) ([]WritePlan, error) {
	if !writable {
		return nil, errors.New("the media source is read-only")
	}
	if strings.TrimSpace(record.Title) == "" {
		return nil, errors.New("select TV metadata before creating NFO plans")
	}
	plans := make([]WritePlan, 0, len(inputs))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, input := range inputs {
		if input.MediaItemID < 1 || strings.TrimSpace(input.TargetPath) == "" {
			return nil, errors.New("TV NFO target is invalid")
		}
		content, buildErr := tvNFOContent(record, input)
		if buildErr != nil {
			return nil, buildErr
		}
		willReplace, inspectErr := inspectNFOReplacement(input.TargetPath)
		if inspectErr != nil {
			return nil, inspectErr
		}
		plan := WritePlan{ID: uuid.NewString(), MediaItemID: input.MediaItemID, TargetPath: input.TargetPath, Content: string(content), State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), WillReplace: willReplace}
		if _, err := tx.ExecContext(ctx, `INSERT INTO write_plans(id,media_item_id,target_path,content,state) VALUES(?,?,?,?,?)`, plan.ID, plan.MediaItemID, plan.TargetPath, plan.Content, plan.State); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('tv.nfo.preview',?,?,?)`, plan.MediaItemID, plan.TargetPath, "TV NFO save plan created"); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.logger.Info("TV NFO plans created", "show_id", record.ShowID, "plan_count", len(plans))
	return plans, nil
}

func tvNFOContent(record TVRecord, input TVNFOInput) ([]byte, error) {
	switch input.Kind {
	case "show":
		return nfo.WriteTVShow(nfo.TVShow{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Genres: record.Genres, TMDbID: record.ProviderID, TVDBID: record.TVDBID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL, Rating: record.Rating, Votes: record.Votes, Status: record.Status, Network: record.Network, Cast: kodiPeople(record.Cast)})
	case "season":
		return nfo.WriteSeason(fmt.Sprintf("Season %d", input.SeasonNumber), input.SeasonNumber)
	case "episode":
		for _, episode := range record.Episodes {
			if episode.SeasonNumber == input.SeasonNumber && episode.EpisodeNumber == input.EpisodeNumber {
				return nfo.WriteEpisode(nfo.Episode{Title: episode.Title, Plot: episode.Overview, Season: episode.SeasonNumber, Episode: episode.EpisodeNumber, AirDate: episode.AirDate, Runtime: episode.RuntimeMinutes, StillURL: episode.StillURL})
			}
		}
		return nil, fmt.Errorf("remote metadata for S%02dE%02d is unavailable", input.SeasonNumber, input.EpisodeNumber)
	default:
		return nil, errors.New("unknown TV NFO kind")
	}
}

func inspectNFOReplacement(target string) (bool, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect NFO target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return true, errors.New("target NFO is not a regular file and cannot be replaced")
	}
	return true, nil
}
func (s *Service) Record(ctx context.Context, itemID int64) (Record, error) {
	return s.repo.GetRecord(ctx, itemID)
}

// ReadExistingNFO loads the sidecar paired with a media file without changing it.
// Symlink sidecars are rejected so an allowlisted media path cannot escape its source.
func (s *Service) ReadExistingNFO(mediaPath string, itemID int64) (Record, bool, error) {
	paths := []string{
		strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".nfo",
		filepath.Join(filepath.Dir(mediaPath), "movie.nfo"),
	}
	var path string
	for _, candidate := range paths {
		if _, err := os.Lstat(candidate); err == nil {
			path = candidate
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return Record{}, false, fmt.Errorf("inspect existing NFO: %w", err)
		}
	}
	if path == "" {
		return Record{}, false, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Record{}, false, fmt.Errorf("inspect existing NFO: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Record{}, false, errors.New("existing NFO must be a regular file")
	}
	if info.Size() > 1<<20 {
		return Record{}, false, errors.New("existing NFO exceeds the 1 MiB safety limit")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return Record{}, false, fmt.Errorf("read existing NFO: %w", err)
	}
	movie, err := nfo.ParseMovie(contents)
	if err != nil {
		return Record{}, false, fmt.Errorf("parse existing NFO: %w", err)
	}
	if movie.Title == "" {
		return Record{}, false, errors.New("existing NFO has no movie title")
	}
	return normalized(Record{MediaItemID: itemID, Provider: "nfo", ProviderID: movie.TMDbID, Title: movie.Title, OriginalTitle: movie.OriginalTitle, Year: movie.Year, Overview: movie.Plot, RuntimeMinutes: movie.Runtime, Genres: movie.Genres, PosterURL: movie.PosterURL, BackdropURL: movie.BackdropURL, Rating: movie.Rating, Votes: movie.Votes, ContentRating: movie.ContentRating, Directors: movie.Directors, Writers: movie.Writers, Studios: movie.Studios, Cast: metadataPeople(movie.Cast), LockedFields: []string{}}), true, nil
}

// HydrateExistingNFO persists local Kodi metadata only when the item has no
// selected or manually saved SQLite metadata. Scans remain read-only for media
// files; this only repopulates the application index after a source is readded.
func (s *Service) HydrateExistingNFO(ctx context.Context, itemID int64, mediaPath string) error {
	existing, err := s.Record(ctx, itemID)
	if err != nil || existing.Title != "" {
		return err
	}
	record, found, err := s.ReadExistingNFO(mediaPath, itemID)
	if err != nil || !found {
		return err
	}
	if _, err := s.Save(ctx, record); err != nil {
		return err
	}
	s.logger.Info("existing movie NFO hydrated", "media_item_id", itemID)
	return nil
}

func normalized(record Record) Record {
	if record.Genres == nil {
		record.Genres = []string{}
	}
	if record.LockedFields == nil {
		record.LockedFields = []string{}
	}
	if record.Directors == nil {
		record.Directors = []string{}
	}
	if record.Writers == nil {
		record.Writers = []string{}
	}
	if record.Studios == nil {
		record.Studios = []string{}
	}
	if record.Cast == nil {
		record.Cast = []Person{}
	}
	return record
}
func kodiPeople(people []Person) []nfo.Person {
	result := make([]nfo.Person, 0, len(people))
	for _, person := range people {
		result = append(result, nfo.Person{Name: person.Name, Role: person.Role, Thumb: person.ProfileURL})
	}
	return result
}
func metadataPeople(people []nfo.Person) []Person {
	result := make([]Person, 0, len(people))
	for _, person := range people {
		result = append(result, Person{Name: person.Name, Role: person.Role, ProfileURL: person.Thumb})
	}
	return result
}
func (s *Service) Save(ctx context.Context, record Record) (Record, error) {
	if strings.TrimSpace(record.Title) == "" {
		return Record{}, errors.New("title is required")
	}
	if err := s.repo.SaveRecord(ctx, record); err != nil {
		return Record{}, err
	}
	return s.Record(ctx, record.MediaItemID)
}

func (s *Service) Preview(ctx context.Context, record Record, mediaPath string, writable bool) (WritePlan, error) {
	if !writable {
		return WritePlan{}, errors.New("the media source is read-only")
	}
	record, err := s.Save(ctx, record)
	if err != nil {
		return WritePlan{}, err
	}
	content, err := nfo.WriteMovie(nfo.Movie{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Runtime: record.RuntimeMinutes, Genres: record.Genres, TMDbID: record.ProviderID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL, Rating: record.Rating, Votes: record.Votes, ContentRating: record.ContentRating, Directors: record.Directors, Writers: record.Writers, Studios: record.Studios, Cast: kodiPeople(record.Cast)})
	if err != nil {
		return WritePlan{}, err
	}
	target, willReplace, err := nfoTarget(mediaPath)
	if err != nil {
		return WritePlan{}, err
	}
	plan := WritePlan{ID: uuid.NewString(), MediaItemID: record.MediaItemID, TargetPath: target, Content: string(content), State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), WillReplace: willReplace}
	if err = s.repo.SaveWritePlan(ctx, plan); err != nil {
		return WritePlan{}, err
	}
	_ = s.repo.InsertAuditEntry(ctx, "nfo.preview", plan.MediaItemID, plan.TargetPath, "NFO save plan created")
	return plan, nil
}
func (s *Service) Plan(ctx context.Context, id string) (WritePlan, error) {
	return s.repo.GetWritePlan(ctx, id)
}
func (s *Service) Apply(ctx context.Context, id string, allowed func(string) bool) (WritePlan, error) {
	plan, err := s.Plan(ctx, id)
	if err != nil {
		return WritePlan{}, err
	}
	if plan.State != "previewed" {
		return WritePlan{}, errors.New("write plan is no longer pending")
	}
	if err := files.CheckAllowed(plan.TargetPath, allowed); err != nil {
		return WritePlan{}, err
	}
	if plan.Conflict {
		_ = s.repo.UpdateWritePlanState(ctx, id, "conflicted")
		return WritePlan{}, errors.New("target NFO is not a regular file and cannot be replaced")
	}
	if err := files.AtomicWriteFile(plan.TargetPath, []byte(plan.Content)); err != nil {
		return WritePlan{}, fmt.Errorf("write NFO: %w", err)
	}
	if err = s.repo.UpdateWritePlanState(ctx, id, "applied"); err != nil {
		return WritePlan{}, err
	}
	detail := "NFO created atomically"
	if plan.WillReplace {
		detail = "Existing NFO replaced atomically"
	}
	_ = s.repo.InsertAuditEntry(ctx, "nfo.apply", plan.MediaItemID, plan.TargetPath, detail)
	return s.Plan(ctx, id)
}

const maxArtworkBytes int64 = 25 << 20

// PreviewArtwork persists a reviewable plan for the selected TMDb poster and
// fanart. It deliberately does not contact the network or write any file.
func (s *Service) PreviewArtwork(ctx context.Context, record Record, mediaPath string, writable bool) (ArtworkPlan, error) {
	if !writable {
		return ArtworkPlan{}, errors.New("the media source is read-only")
	}
	for _, item := range []struct{ kind, source string }{{"poster", record.PosterURL}, {"fanart", record.BackdropURL}} {
		if strings.TrimSpace(item.source) != "" {
			if err := validateTMDbImageURL(item.source); err != nil {
				return ArtworkPlan{}, fmt.Errorf("%s image: %w", item.kind, err)
			}
		}
	}
	if record.PosterURL == "" && record.BackdropURL == "" {
		return ArtworkPlan{}, errors.New("select metadata with artwork before creating download plans")
	}
	directory := filepath.Dir(mediaPath)
	assets := make([]ArtworkAsset, 0, 2)
	if record.PosterURL != "" {
		target := filepath.Join(directory, "poster.jpg")
		willReplace, conflict, err := files.ValidateTarget(target)
		if err != nil {
			return ArtworkPlan{}, err
		}
		assets = append(assets, ArtworkAsset{Kind: "poster", SourceURL: record.PosterURL, TargetPath: target, Conflict: conflict, WillReplace: willReplace})
	}
	if record.BackdropURL != "" {
		target := filepath.Join(directory, "fanart.jpg")
		willReplace, conflict, err := files.ValidateTarget(target)
		if err != nil {
			return ArtworkPlan{}, err
		}
		assets = append(assets, ArtworkAsset{Kind: "fanart", SourceURL: record.BackdropURL, TargetPath: target, Conflict: conflict, WillReplace: willReplace})
	}
	plan := ArtworkPlan{ID: uuid.NewString(), MediaItemID: record.MediaItemID, State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), Assets: assets}
	if err := s.repo.SaveArtworkPlan(ctx, plan); err != nil {
		return ArtworkPlan{}, err
	}
	_ = s.repo.InsertAuditEntry(ctx, "artwork.preview", plan.MediaItemID, filepath.Dir(mediaPath), "Artwork download plan created")
	s.logger.Info("Artwork download plan created", "media_item_id", plan.MediaItemID, "asset_count", len(plan.Assets))
	return plan, nil
}

func (s *Service) ArtworkPlan(ctx context.Context, id string) (ArtworkPlan, error) {
	return s.repo.GetArtworkPlan(ctx, id)
}

// ArtworkCandidates refreshes the persisted Fanart.tv candidate catalog for a
// matched movie. The HTTP layer only receives normalized candidates, never a
// provider response payload.
func (s *Service) ArtworkCandidates(ctx context.Context, itemID int64) ([]ArtworkCandidate, error) {
	record, err := s.Record(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if record.Provider != "tmdb" || strings.TrimSpace(record.ProviderID) == "" {
		return nil, errors.New("select TMDb metadata before loading artwork candidates")
	}
	if s.fanart == nil {
		return nil, errors.New("Fanart.tv provider is unavailable")
	}
	assets, err := s.fanart.Movie(ctx, record.ProviderID)
	if err != nil {
		return nil, err
	}
	candidates := make([]ArtworkCandidate, 0, len(assets))
	for _, asset := range assets {
		if strings.TrimSpace(asset.ID) == "" || strings.TrimSpace(asset.URL) == "" {
			continue
		}
		previewURL, err := artwork.FanartPreviewURL(asset.URL)
		if err != nil {
			continue
		}
		candidates = append(candidates, ArtworkCandidate{
			ID: fmt.Sprintf("fanart:%d:%s:%s", itemID, asset.Kind, asset.ID), MediaItemID: itemID,
			Provider: "fanart.tv", ProviderAssetID: asset.ID, Kind: asset.Kind, SourceURL: asset.URL,
			PreviewURL: previewURL, Language: asset.Lang, Likes: asset.Likes,
			Width: asset.Width, Height: asset.Height, MimeType: imageMIME(asset.URL),
		})
	}
	if err := s.repo.ReplaceArtworkCandidates(ctx, candidates); err != nil {
		return nil, err
	}
	return s.repo.ListArtworkCandidates(ctx, itemID)
}

func (s *Service) CachedArtworkCandidates(ctx context.Context, itemID int64) ([]ArtworkCandidate, error) {
	return s.repo.ListArtworkCandidates(ctx, itemID)
}

// OpenArtworkPreview resolves a persisted candidate and opens its reduced
// preview through the configured server-side HTTP client. Callers must close
// Body; the candidate ID is the only client-controlled lookup value.
func (s *Service) OpenArtworkPreview(ctx context.Context, itemID int64, candidateID string) (ArtworkPreview, error) {
	candidate, err := s.repo.GetArtworkCandidate(ctx, candidateID)
	if err != nil || candidate.MediaItemID != itemID {
		return ArtworkPreview{}, errors.New("artwork candidate not found")
	}
	return s.openArtworkPreview(ctx, candidate.Provider, candidate.PreviewURL)
}

func (s *Service) openArtworkPreview(ctx context.Context, provider, previewURL string) (ArtworkPreview, error) {
	if err := validateArtworkSource(provider, previewURL); err != nil {
		return ArtworkPreview{}, err
	}
	s.artworkMu.RLock()
	client := s.artworkClient
	s.artworkMu.RUnlock()
	if client == nil {
		return ArtworkPreview{}, errors.New("artwork client is unavailable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, previewURL, nil)
	if err != nil {
		return ArtworkPreview{}, errors.New("create artwork preview request")
	}
	response, err := client.Do(request)
	if err != nil {
		return ArtworkPreview{}, errors.New("download artwork preview request failed")
	}
	if response.Request != nil && response.Request.URL.String() != request.URL.String() {
		response.Body.Close()
		return ArtworkPreview{}, errors.New("artwork preview redirect is not allowed")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		response.Body.Close()
		return ArtworkPreview{}, fmt.Errorf("artwork preview provider returned HTTP %d", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (contentType != "image/jpeg" && contentType != "image/png") {
		response.Body.Close()
		return ArtworkPreview{}, errors.New("artwork preview provider did not return a JPEG or PNG image")
	}
	if response.ContentLength > artwork.MaxArtworkBytes {
		response.Body.Close()
		return ArtworkPreview{}, errors.New("artwork preview exceeds the 25 MiB safety limit")
	}
	reader := bufio.NewReader(io.LimitReader(response.Body, artwork.MaxArtworkBytes+1))
	header, err := reader.Peek(3)
	validJPEG := err == nil && len(header) == 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff
	validPNG := false
	if contentType == "image/png" {
		pngHeader, pngErr := reader.Peek(8)
		validPNG = pngErr == nil && string(pngHeader) == "\x89PNG\r\n\x1a\n"
	}
	if (contentType == "image/jpeg" && !validJPEG) || (contentType == "image/png" && !validPNG) {
		response.Body.Close()
		return ArtworkPreview{}, errors.New("artwork preview provider returned an invalid image")
	}
	return ArtworkPreview{Body: structReadCloser{Reader: reader, closer: response.Body}, ContentType: contentType, ContentLength: response.ContentLength}, nil
}

type structReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r structReadCloser) Close() error { return r.closer.Close() }

func (s *Service) PreviewArtworkSelection(ctx context.Context, itemID int64, selections []ArtworkSelection, mediaPath string, writable bool) (ArtworkPlan, error) {
	if !writable {
		return ArtworkPlan{}, errors.New("the media source is read-only")
	}
	if len(selections) == 0 {
		return ArtworkPlan{}, errors.New("select at least one artwork candidate")
	}
	assets := make([]ArtworkAsset, 0, len(selections))
	seenKinds := make(map[string]struct{}, len(selections))
	directory := filepath.Dir(mediaPath)
	for _, selection := range selections {
		if !supportedArtworkKind(selection.Kind) {
			return ArtworkPlan{}, fmt.Errorf("unsupported artwork kind %q", selection.Kind)
		}
		if _, ok := seenKinds[selection.Kind]; ok {
			return ArtworkPlan{}, fmt.Errorf("multiple candidates selected for %s", selection.Kind)
		}
		seenKinds[selection.Kind] = struct{}{}
		candidate, err := s.repo.GetArtworkCandidate(ctx, selection.CandidateID)
		if err != nil || candidate.MediaItemID != itemID || candidate.Kind != selection.Kind {
			return ArtworkPlan{}, fmt.Errorf("artwork candidate %q is not available for this movie", selection.CandidateID)
		}
		if err := validateArtworkSource(candidate.Provider, candidate.SourceURL); err != nil {
			return ArtworkPlan{}, fmt.Errorf("%s image: %w", candidate.Kind, err)
		}
		extension := ".jpg"
		if candidate.MimeType == "image/png" {
			extension = ".png"
		}
		target := filepath.Join(directory, candidate.Kind+extension)
		willReplace, conflict, err := files.ValidateTarget(target)
		if err != nil {
			return ArtworkPlan{}, err
		}
		assets = append(assets, ArtworkAsset{Kind: candidate.Kind, CandidateID: candidate.ID, Provider: candidate.Provider, ProviderAssetID: candidate.ProviderAssetID, SourceURL: candidate.SourceURL, PreviewURL: candidate.PreviewURL, Language: candidate.Language, Likes: candidate.Likes, Width: candidate.Width, Height: candidate.Height, MimeType: candidate.MimeType, TargetPath: target, Conflict: conflict, WillReplace: willReplace})
	}
	plan := ArtworkPlan{ID: uuid.NewString(), MediaItemID: itemID, State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), Assets: assets}
	if err := s.repo.SaveArtworkPlan(ctx, plan); err != nil {
		return ArtworkPlan{}, err
	}
	_ = s.repo.InsertAuditEntry(ctx, "artwork.preview", itemID, directory, "Fanart.tv artwork selection plan created")
	return plan, nil
}

func (s *Service) QueueArtwork(ctx context.Context, id string, allowed func(string) bool) (ArtworkPlan, error) {
	if s.jobQueue == nil {
		return ArtworkPlan{}, errors.New("artwork job service is unavailable")
	}
	plan, err := s.ArtworkPlan(ctx, id)
	if err != nil {
		return ArtworkPlan{}, err
	}
	if plan.State != "previewed" {
		return ArtworkPlan{}, errors.New("artwork plan is no longer pending")
	}
	for _, asset := range plan.Assets {
		if err := files.CheckAllowed(asset.TargetPath, allowed); err != nil {
			return ArtworkPlan{}, errors.New("artwork target is outside configured media roots")
		}
		if asset.Conflict {
			_ = s.repo.UpdateArtworkPlanState(ctx, id, "conflicted")
			return ArtworkPlan{}, fmt.Errorf("%s target is not a regular file and cannot be replaced", asset.Kind)
		}
	}
	payload, _ := json.Marshal(map[string]string{"planId": id})
	if _, err := s.jobQueue.QueuePayload(ctx, "artwork_download", nil, payload); err != nil {
		return ArtworkPlan{}, err
	}
	if err := s.repo.UpdateArtworkPlanState(ctx, id, "queued"); err != nil {
		return ArtworkPlan{}, err
	}
	return s.ArtworkPlan(ctx, id)
}

func (s *Service) ApplyArtwork(ctx context.Context, id string, allowed func(string) bool) (ArtworkPlan, error) {
	plan, err := s.ArtworkPlan(ctx, id)
	if err != nil {
		return ArtworkPlan{}, err
	}
	if plan.State != "previewed" {
		return ArtworkPlan{}, errors.New("artwork plan is no longer pending")
	}
	for _, asset := range plan.Assets {
		if err := files.CheckAllowed(asset.TargetPath, allowed); err != nil {
			return ArtworkPlan{}, errors.New("artwork target is outside configured media roots")
		}
		if asset.Conflict {
			_ = s.repo.UpdateArtworkPlanState(ctx, id, "conflicted")
			return ArtworkPlan{}, fmt.Errorf("%s target is not a regular file and cannot be replaced", asset.Kind)
		}
		if err := validateArtworkSource(asset.Provider, asset.SourceURL); err != nil {
			return ArtworkPlan{}, fmt.Errorf("%s image: %w", asset.Kind, err)
		}
	}
	return s.applyArtworkFiles(ctx, plan, func(int, string) {})
}

func (s *Service) handleArtworkDownloadJob(ctx context.Context, job jobs.Job, updateProgress func(int, string)) error {
	var payload struct {
		PlanID string `json:"planId"`
	}
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.PlanID == "" {
		return errors.New("artwork job payload is invalid")
	}
	plan, err := s.ArtworkPlan(ctx, payload.PlanID)
	if err != nil {
		return err
	}
	if plan.State != "queued" && plan.State != "running" {
		return errors.New("artwork plan is not queued")
	}
	if err := s.repo.UpdateArtworkPlanState(ctx, plan.ID, "running"); err != nil {
		return err
	}
	_, err = s.applyArtworkFiles(ctx, plan, updateProgress)
	if err != nil && job.RetryCount < job.MaxRetries && !errors.Is(err, context.Canceled) {
		if current, getErr := s.ArtworkPlan(ctx, plan.ID); getErr == nil && current.State == "failed" {
			_ = s.repo.UpdateArtworkPlanState(ctx, plan.ID, "queued")
		}
	}
	return err
}

func (s *Service) applyArtworkFiles(ctx context.Context, plan ArtworkPlan, updateProgress func(int, string)) (ArtworkPlan, error) {
	lockValue, _ := s.artworkLocks.LoadOrStore(plan.MediaItemID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	s.logger.Info("Artwork download started", "media_item_id", plan.MediaItemID, "asset_count", len(plan.Assets))
	temps := make([]string, 0, len(plan.Assets))
	defer func() {
		for _, temp := range temps {
			_ = os.Remove(temp)
		}
	}()
	s.artworkMu.RLock()
	client := s.artworkClient
	s.artworkMu.RUnlock()
	for index, asset := range plan.Assets {
		updateProgress(index, "Downloading "+asset.Kind)
		temp, downloadErr := downloadArtwork(ctx, client, asset)
		if downloadErr != nil {
			_ = s.repo.UpdateArtworkPlanState(ctx, plan.ID, "failed")
			s.logger.Warn("Artwork download failed", "media_item_id", plan.MediaItemID, "asset_kind", asset.Kind, "error", downloadErr)
			return ArtworkPlan{}, downloadErr
		}
		temps = append(temps, temp)
	}
	for index, asset := range plan.Assets {
		willReplace, conflict, inspectErr := files.ValidateTarget(asset.TargetPath)
		if inspectErr != nil || conflict {
			_ = s.repo.UpdateArtworkPlanState(ctx, plan.ID, "conflicted")
			return ArtworkPlan{}, fmt.Errorf("%s target changed and cannot be replaced", asset.Kind)
		}
		if err := os.Rename(temps[index], asset.TargetPath); err != nil {
			_ = s.repo.UpdateArtworkPlanState(ctx, plan.ID, "failed")
			return ArtworkPlan{}, fmt.Errorf("replace %s artwork: %w", asset.Kind, err)
		}
		if err := s.repo.SetArtworkAsset(ctx, asset, plan.MediaItemID); err != nil {
			return ArtworkPlan{}, err
		}
		_ = willReplace
		updateProgress(index+1, "Saved "+asset.Kind)
	}
	if err := s.repo.UpdateArtworkPlanState(ctx, plan.ID, "applied"); err != nil {
		return ArtworkPlan{}, err
	}
	for _, asset := range plan.Assets {
		detail := "Artwork created atomically"
		if asset.WillReplace {
			detail = "Existing artwork replaced atomically"
		}
		_ = s.repo.InsertAuditEntry(ctx, "artwork.apply", plan.MediaItemID, asset.TargetPath, detail)
	}
	s.logger.Info("Artwork download completed", "media_item_id", plan.MediaItemID, "asset_count", len(plan.Assets))
	return s.ArtworkPlan(ctx, plan.ID)
}

func downloadArtwork(ctx context.Context, client *http.Client, asset ArtworkAsset) (string, error) {
	return artwork.DownloadImage(ctx, client, asset.SourceURL, filepath.Dir(asset.TargetPath), asset.MimeType)
}

func validateTMDbImageURL(value string) error {
	return artwork.ValidateTMDbImageURL(value)
}

var artworkKinds = map[string]bool{"poster": true, "fanart": true, "clearlogo": true, "clearart": true, "discart": true, "banner": true, "landscape": true}

func supportedArtworkKind(kind string) bool { return artworkKinds[kind] }

func validateArtworkSource(provider, value string) error {
	if provider == "fanart.tv" {
		return artwork.ValidateFanartImageURL(value)
	}
	return artwork.ValidateTMDbImageURL(value)
}

func imageMIME(value string) string {
	if strings.HasSuffix(strings.ToLower(strings.Split(value, "?")[0]), ".png") {
		return "image/png"
	}
	return "image/jpeg"
}

func nfoTarget(mediaPath string) (string, bool, error) {
	candidates := []string{strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".nfo", filepath.Join(filepath.Dir(mediaPath), "movie.nfo")}
	for _, candidate := range candidates {
		willReplace, _, err := files.ValidateTarget(candidate)
		if err == nil && willReplace {
			return candidate, true, nil
		} else if err != nil {
			return "", false, fmt.Errorf("inspect NFO target: %w", err)
		}
	}
	return candidates[0], false, nil
}
