package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/artwork"
	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/nfo"
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
	Kind        string `json:"kind"`
	SourceURL   string `json:"sourceUrl"`
	TargetPath  string `json:"targetPath"`
	Conflict    bool   `json:"conflict"`
	WillReplace bool   `json:"willReplace"`
}
type ArtworkPlan struct {
	ID          string         `json:"id"`
	MediaItemID int64          `json:"mediaItemId"`
	State       string         `json:"state"`
	CreatedAt   string         `json:"createdAt"`
	Assets      []ArtworkAsset `json:"assets"`
}
type TVRecord struct {
	ShowID        int64              `json:"showId"`
	Provider      string             `json:"provider"`
	ProviderID    string             `json:"providerId"`
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
	logger        *slog.Logger
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
	episodes := make([]TVEpisodeDetails, 0)
	for _, season := range seasons {
		remote, fetchErr := provider.TVSeason(ctx, providerID, season)
		if fetchErr != nil {
			s.logger.Warn("TMDb TV season lookup failed", "show_id", showID, "season_number", season, "error", fetchErr)
			return TVRecord{}, fetchErr
		}
		episodes = append(episodes, remote...)
	}
	record := TVRecord{ShowID: showID, Provider: "tmdb", ProviderID: details.ID, Title: details.Title, OriginalTitle: details.OriginalTitle, Year: details.Year, Overview: details.Overview, Genres: details.Genres, PosterURL: details.PosterURL, BackdropURL: details.BackdropURL, Rating: details.Rating, Votes: details.Votes, Status: details.Status, Network: details.Network, Cast: details.Cast, Episodes: episodes}
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
			record.ProviderID, record.Genres, record.PosterURL, record.BackdropURL = show.TMDbID, show.Genres, show.PosterURL, show.BackdropURL
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
		return nfo.WriteTVShow(nfo.TVShow{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Genres: record.Genres, TMDbID: record.ProviderID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL, Rating: record.Rating, Votes: record.Votes, Status: record.Status, Network: record.Network, Cast: kodiPeople(record.Cast)})
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
		if err := validateTMDbImageURL(asset.SourceURL); err != nil {
			return ArtworkPlan{}, fmt.Errorf("%s image: %w", asset.Kind, err)
		}
	}
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
	for _, asset := range plan.Assets {
		temp, downloadErr := downloadArtwork(ctx, client, asset)
		if downloadErr != nil {
			_ = s.repo.UpdateArtworkPlanState(ctx, id, "failed")
			s.logger.Warn("Artwork download failed", "media_item_id", plan.MediaItemID, "asset_kind", asset.Kind, "error", downloadErr)
			return ArtworkPlan{}, downloadErr
		}
		temps = append(temps, temp)
	}
	for index, asset := range plan.Assets {
		if err := os.Rename(temps[index], asset.TargetPath); err != nil {
			_ = s.repo.UpdateArtworkPlanState(ctx, id, "failed")
			return ArtworkPlan{}, fmt.Errorf("replace %s artwork: %w", asset.Kind, err)
		}
	}
	if err := s.repo.UpdateArtworkPlanState(ctx, id, "applied"); err != nil {
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
	return s.ArtworkPlan(ctx, id)
}

func downloadArtwork(ctx context.Context, client *http.Client, asset ArtworkAsset) (string, error) {
	return artwork.DownloadJPEG(ctx, client, asset.SourceURL, filepath.Dir(asset.TargetPath))
}

func validateTMDbImageURL(value string) error {
	return artwork.ValidateTMDbImageURL(value)
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
