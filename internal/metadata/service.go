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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/kodi"
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
	provider      Provider
	artworkMu     sync.RWMutex
	artworkClient *http.Client
	logger        *slog.Logger
}

func NewService(db *sql.DB, provider Provider) *Service {
	service := &Service{db: db, provider: provider, artworkClient: http.DefaultClient, logger: slog.Default()}
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

func (s *Service) TVRecord(ctx context.Context, showID int64) (TVRecord, error) {
	var record TVRecord
	var year, votes sql.NullInt64
	var rating sql.NullFloat64
	var genres, cast string
	err := s.db.QueryRowContext(ctx, `SELECT show_id,provider,provider_id,title,original_title,year,overview,genres_json,poster_url,backdrop_url,rating,votes,status,network,cast_json,updated_at FROM tv_metadata WHERE show_id=?`, showID).Scan(&record.ShowID, &record.Provider, &record.ProviderID, &record.Title, &record.OriginalTitle, &year, &record.Overview, &genres, &record.PosterURL, &record.BackdropURL, &rating, &votes, &record.Status, &record.Network, &cast, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return normalizedTV(TVRecord{ShowID: showID}), nil
	}
	if err != nil {
		return TVRecord{}, err
	}
	if year.Valid {
		value := int(year.Int64)
		record.Year = &value
	}
	if rating.Valid {
		value := rating.Float64
		record.Rating = &value
	}
	if votes.Valid {
		value := int(votes.Int64)
		record.Votes = &value
	}
	_ = json.Unmarshal([]byte(genres), &record.Genres)
	_ = json.Unmarshal([]byte(cast), &record.Cast)
	rows, err := s.db.QueryContext(ctx, `SELECT season_number,episode_number,title,overview,air_date,runtime_minutes,still_url FROM tv_episode_metadata WHERE show_id=? ORDER BY season_number,episode_number`, showID)
	if err != nil {
		return TVRecord{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var episode TVEpisodeDetails
		var runtime sql.NullInt64
		if err := rows.Scan(&episode.SeasonNumber, &episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.AirDate, &runtime, &episode.StillURL); err != nil {
			return TVRecord{}, err
		}
		if runtime.Valid {
			value := int(runtime.Int64)
			episode.RuntimeMinutes = &value
		}
		record.Episodes = append(record.Episodes, episode)
	}
	return normalizedTV(record), rows.Err()
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
			show, err := kodi.ParseTVShow(contents)
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
			episode, err := kodi.ParseEpisode(contents)
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
	record = normalizedTV(record)
	genres, _ := json.Marshal(record.Genres)
	cast, _ := json.Marshal(record.Cast)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TVRecord{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO tv_metadata(show_id,provider,provider_id,title,original_title,year,overview,genres_json,poster_url,backdrop_url,rating,votes,status,network,cast_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now')) ON CONFLICT(show_id) DO UPDATE SET provider=excluded.provider,provider_id=excluded.provider_id,title=excluded.title,original_title=excluded.original_title,year=excluded.year,overview=excluded.overview,genres_json=excluded.genres_json,poster_url=excluded.poster_url,backdrop_url=excluded.backdrop_url,rating=excluded.rating,votes=excluded.votes,status=excluded.status,network=excluded.network,cast_json=excluded.cast_json,updated_at=datetime('now')`, record.ShowID, record.Provider, record.ProviderID, strings.TrimSpace(record.Title), strings.TrimSpace(record.OriginalTitle), record.Year, strings.TrimSpace(record.Overview), string(genres), record.PosterURL, record.BackdropURL, record.Rating, record.Votes, strings.TrimSpace(record.Status), strings.TrimSpace(record.Network), string(cast))
	if err != nil {
		return TVRecord{}, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM tv_episode_metadata WHERE show_id=?`, record.ShowID); err != nil {
		return TVRecord{}, err
	}
	for _, episode := range record.Episodes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO tv_episode_metadata(show_id,season_number,episode_number,title,overview,air_date,runtime_minutes,still_url) VALUES(?,?,?,?,?,?,?,?)`, record.ShowID, episode.SeasonNumber, episode.EpisodeNumber, strings.TrimSpace(episode.Title), strings.TrimSpace(episode.Overview), strings.TrimSpace(episode.AirDate), episode.RuntimeMinutes, episode.StillURL); err != nil {
			return TVRecord{}, err
		}
	}
	if err = tx.Commit(); err != nil {
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
		return kodi.WriteTVShow(kodi.TVShow{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Genres: record.Genres, TMDbID: record.ProviderID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL, Rating: record.Rating, Votes: record.Votes, Status: record.Status, Network: record.Network, Cast: kodiPeople(record.Cast)})
	case "season":
		return kodi.WriteSeason(fmt.Sprintf("Season %d", input.SeasonNumber), input.SeasonNumber)
	case "episode":
		for _, episode := range record.Episodes {
			if episode.SeasonNumber == input.SeasonNumber && episode.EpisodeNumber == input.EpisodeNumber {
				return kodi.WriteEpisode(kodi.Episode{Title: episode.Title, Plot: episode.Overview, Season: episode.SeasonNumber, Episode: episode.EpisodeNumber, AirDate: episode.AirDate, Runtime: episode.RuntimeMinutes, StillURL: episode.StillURL})
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
	var record Record
	var year, runtime sql.NullInt64
	var genres, directors, writers, studios, cast, locked string
	var rating sql.NullFloat64
	var votes sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,rating,votes,content_rating,directors_json,writers_json,studios_json,cast_json,locked_fields_json,updated_at FROM media_metadata WHERE media_item_id=?`, itemID).Scan(&record.MediaItemID, &record.Provider, &record.ProviderID, &record.Title, &record.OriginalTitle, &year, &record.Overview, &runtime, &genres, &record.PosterURL, &record.BackdropURL, &rating, &votes, &record.ContentRating, &directors, &writers, &studios, &cast, &locked, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return normalized(Record{MediaItemID: itemID, Genres: []string{}, LockedFields: []string{}}), nil
	}
	if err != nil {
		return Record{}, err
	}
	if year.Valid {
		value := int(year.Int64)
		record.Year = &value
	}
	if runtime.Valid {
		value := int(runtime.Int64)
		record.RuntimeMinutes = &value
	}
	if rating.Valid {
		value := rating.Float64
		record.Rating = &value
	}
	if votes.Valid {
		value := int(votes.Int64)
		record.Votes = &value
	}
	_ = json.Unmarshal([]byte(genres), &record.Genres)
	_ = json.Unmarshal([]byte(directors), &record.Directors)
	_ = json.Unmarshal([]byte(writers), &record.Writers)
	_ = json.Unmarshal([]byte(studios), &record.Studios)
	_ = json.Unmarshal([]byte(cast), &record.Cast)
	_ = json.Unmarshal([]byte(locked), &record.LockedFields)
	return normalized(record), nil
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
	movie, err := kodi.ParseMovie(contents)
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
func kodiPeople(people []Person) []kodi.Person {
	result := make([]kodi.Person, 0, len(people))
	for _, person := range people {
		result = append(result, kodi.Person{Name: person.Name, Role: person.Role, Thumb: person.ProfileURL})
	}
	return result
}
func metadataPeople(people []kodi.Person) []Person {
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
	record = normalized(record)
	genres, _ := json.Marshal(record.Genres)
	directors, _ := json.Marshal(record.Directors)
	writers, _ := json.Marshal(record.Writers)
	studios, _ := json.Marshal(record.Studios)
	cast, _ := json.Marshal(record.Cast)
	locked, _ := json.Marshal(record.LockedFields)
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_metadata(media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,rating,votes,content_rating,directors_json,writers_json,studios_json,cast_json,locked_fields_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now')) ON CONFLICT(media_item_id) DO UPDATE SET provider=excluded.provider,provider_id=excluded.provider_id,title=excluded.title,original_title=excluded.original_title,year=excluded.year,overview=excluded.overview,runtime_minutes=excluded.runtime_minutes,genres_json=excluded.genres_json,poster_url=excluded.poster_url,backdrop_url=excluded.backdrop_url,rating=excluded.rating,votes=excluded.votes,content_rating=excluded.content_rating,directors_json=excluded.directors_json,writers_json=excluded.writers_json,studios_json=excluded.studios_json,cast_json=excluded.cast_json,locked_fields_json=excluded.locked_fields_json,updated_at=datetime('now')`, record.MediaItemID, record.Provider, record.ProviderID, strings.TrimSpace(record.Title), strings.TrimSpace(record.OriginalTitle), record.Year, strings.TrimSpace(record.Overview), record.RuntimeMinutes, string(genres), record.PosterURL, record.BackdropURL, record.Rating, record.Votes, strings.TrimSpace(record.ContentRating), string(directors), string(writers), string(studios), string(cast), string(locked))
	if err != nil {
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
	content, err := kodi.WriteMovie(kodi.Movie{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Runtime: record.RuntimeMinutes, Genres: record.Genres, TMDbID: record.ProviderID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL, Rating: record.Rating, Votes: record.Votes, ContentRating: record.ContentRating, Directors: record.Directors, Writers: record.Writers, Studios: record.Studios, Cast: kodiPeople(record.Cast)})
	if err != nil {
		return WritePlan{}, err
	}
	target, willReplace, err := nfoTarget(mediaPath)
	if err != nil {
		return WritePlan{}, err
	}
	plan := WritePlan{ID: uuid.NewString(), MediaItemID: record.MediaItemID, TargetPath: target, Content: string(content), State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), WillReplace: willReplace}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO write_plans(id,media_item_id,target_path,content,state) VALUES(?,?,?,?,?)`, plan.ID, plan.MediaItemID, plan.TargetPath, plan.Content, plan.State); err != nil {
		return WritePlan{}, err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('nfo.preview',?,?,?)`, plan.MediaItemID, plan.TargetPath, "NFO save plan created")
	return plan, nil
}
func (s *Service) Plan(ctx context.Context, id string) (WritePlan, error) {
	var plan WritePlan
	err := s.db.QueryRowContext(ctx, `SELECT id,media_item_id,target_path,content,state,created_at FROM write_plans WHERE id=?`, id).Scan(&plan.ID, &plan.MediaItemID, &plan.TargetPath, &plan.Content, &plan.State, &plan.CreatedAt)
	if err != nil {
		return WritePlan{}, errors.New("write plan not found")
	}
	info, statErr := os.Lstat(plan.TargetPath)
	plan.WillReplace = statErr == nil
	plan.Conflict = statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular())
	return plan, nil
}
func (s *Service) Apply(ctx context.Context, id string, allowed func(string) bool) (WritePlan, error) {
	plan, err := s.Plan(ctx, id)
	if err != nil {
		return WritePlan{}, err
	}
	if plan.State != "previewed" {
		return WritePlan{}, errors.New("write plan is no longer pending")
	}
	if !allowed(plan.TargetPath) {
		return WritePlan{}, errors.New("target path is outside configured media roots")
	}
	if plan.Conflict {
		_, _ = s.db.ExecContext(ctx, `UPDATE write_plans SET state='conflicted' WHERE id=?`, id)
		return WritePlan{}, errors.New("target NFO is not a regular file and cannot be replaced")
	}
	directory := filepath.Dir(plan.TargetPath)
	temp, err := os.CreateTemp(directory, ".mediagrap-*.nfo")
	if err != nil {
		return WritePlan{}, err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err = temp.WriteString(plan.Content); err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return WritePlan{}, fmt.Errorf("write NFO: %w", err)
	}
	if err = os.Rename(tempName, plan.TargetPath); err != nil {
		return WritePlan{}, fmt.Errorf("replace NFO: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE write_plans SET state='applied',applied_at=datetime('now') WHERE id=?`, id)
	if err != nil {
		return WritePlan{}, err
	}
	detail := "NFO created atomically"
	if plan.WillReplace {
		detail = "Existing NFO replaced atomically"
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('nfo.apply',?,?,?)`, plan.MediaItemID, plan.TargetPath, detail)
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
	record, err := s.Save(ctx, record)
	if err != nil {
		return ArtworkPlan{}, err
	}
	assets := make([]ArtworkAsset, 0, 2)
	for _, item := range []struct{ kind, source, filename string }{
		{"poster", record.PosterURL, "poster.jpg"},
		{"fanart", record.BackdropURL, "fanart.jpg"},
	} {
		if strings.TrimSpace(item.source) == "" {
			continue
		}
		target := filepath.Join(filepath.Dir(mediaPath), item.filename)
		asset := ArtworkAsset{Kind: item.kind, SourceURL: item.source, TargetPath: target}
		if info, statErr := os.Lstat(target); statErr == nil {
			asset.WillReplace = true
			asset.Conflict = info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return ArtworkPlan{}, fmt.Errorf("inspect %s target: %w", item.kind, statErr)
		}
		assets = append(assets, asset)
	}
	if len(assets) == 0 {
		return ArtworkPlan{}, errors.New("select a TMDb poster or fanart before creating an artwork plan")
	}
	plan := ArtworkPlan{ID: uuid.NewString(), MediaItemID: record.MediaItemID, State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), Assets: assets}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ArtworkPlan{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO artwork_plans(id,media_item_id,state) VALUES(?,?,?)`, plan.ID, plan.MediaItemID, plan.State); err != nil {
		return ArtworkPlan{}, err
	}
	for _, asset := range assets {
		if _, err = tx.ExecContext(ctx, `INSERT INTO artwork_plan_assets(artwork_plan_id,kind,source_url,target_path) VALUES(?,?,?,?)`, plan.ID, asset.Kind, asset.SourceURL, asset.TargetPath); err != nil {
			return ArtworkPlan{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('artwork.preview',?,?,?)`, plan.MediaItemID, filepath.Dir(mediaPath), "Artwork download plan created"); err != nil {
		return ArtworkPlan{}, err
	}
	if err = tx.Commit(); err != nil {
		return ArtworkPlan{}, err
	}
	s.logger.Info("Artwork download plan created", "media_item_id", plan.MediaItemID, "asset_count", len(plan.Assets))
	return plan, nil
}

func (s *Service) ArtworkPlan(ctx context.Context, id string) (ArtworkPlan, error) {
	var plan ArtworkPlan
	err := s.db.QueryRowContext(ctx, `SELECT id,media_item_id,state,created_at FROM artwork_plans WHERE id=?`, id).Scan(&plan.ID, &plan.MediaItemID, &plan.State, &plan.CreatedAt)
	if err != nil {
		return ArtworkPlan{}, errors.New("artwork plan not found")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT kind,source_url,target_path FROM artwork_plan_assets WHERE artwork_plan_id=? ORDER BY id`, id)
	if err != nil {
		return ArtworkPlan{}, err
	}
	defer rows.Close()
	plan.Assets = []ArtworkAsset{}
	for rows.Next() {
		var asset ArtworkAsset
		if err := rows.Scan(&asset.Kind, &asset.SourceURL, &asset.TargetPath); err != nil {
			return ArtworkPlan{}, err
		}
		if info, statErr := os.Lstat(asset.TargetPath); statErr == nil {
			asset.WillReplace = true
			asset.Conflict = info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()
		}
		plan.Assets = append(plan.Assets, asset)
	}
	return plan, rows.Err()
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
		if !allowed(asset.TargetPath) {
			return ArtworkPlan{}, errors.New("artwork target is outside configured media roots")
		}
		if asset.Conflict {
			_, _ = s.db.ExecContext(ctx, `UPDATE artwork_plans SET state='conflicted' WHERE id=?`, id)
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
			_, _ = s.db.ExecContext(ctx, `UPDATE artwork_plans SET state='failed' WHERE id=?`, id)
			s.logger.Warn("Artwork download failed", "media_item_id", plan.MediaItemID, "asset_kind", asset.Kind, "error", downloadErr)
			return ArtworkPlan{}, downloadErr
		}
		temps = append(temps, temp)
	}
	for index, asset := range plan.Assets {
		if err := os.Rename(temps[index], asset.TargetPath); err != nil {
			_, _ = s.db.ExecContext(ctx, `UPDATE artwork_plans SET state='failed' WHERE id=?`, id)
			return ArtworkPlan{}, fmt.Errorf("replace %s artwork: %w", asset.Kind, err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE artwork_plans SET state='applied',applied_at=datetime('now') WHERE id=?`, id); err != nil {
		return ArtworkPlan{}, err
	}
	for _, asset := range plan.Assets {
		detail := "Artwork created atomically"
		if asset.WillReplace {
			detail = "Existing artwork replaced atomically"
		}
		_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('artwork.apply',?,?,?)`, plan.MediaItemID, asset.TargetPath, detail)
	}
	s.logger.Info("Artwork download completed", "media_item_id", plan.MediaItemID, "asset_count", len(plan.Assets))
	return s.ArtworkPlan(ctx, id)
}

func downloadArtwork(ctx context.Context, client *http.Client, asset ArtworkAsset) (string, error) {
	if client == nil {
		return "", errors.New("artwork client is unavailable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.SourceURL, nil)
	if err != nil {
		return "", errors.New("create artwork request")
	}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("download artwork request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("artwork provider returned HTTP %d", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "image/jpeg" {
		return "", errors.New("artwork provider did not return a JPEG image")
	}
	reader := bufio.NewReader(response.Body)
	header, err := reader.Peek(3)
	if err != nil || len(header) != 3 || header[0] != 0xff || header[1] != 0xd8 || header[2] != 0xff {
		return "", errors.New("artwork provider returned an invalid JPEG image")
	}
	temp, err := os.CreateTemp(filepath.Dir(asset.TargetPath), ".mediagrap-*.jpg")
	if err != nil {
		return "", err
	}
	tempName := temp.Name()
	written, copyErr := io.Copy(temp, io.LimitReader(reader, maxArtworkBytes+1))
	if copyErr == nil && written > maxArtworkBytes {
		copyErr = errors.New("artwork exceeds the 25 MiB safety limit")
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("write artwork: %w", copyErr)
	}
	return tempName, nil
}

func validateTMDbImageURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "image.tmdb.org" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.EscapedPath(), "/t/p/") {
		return errors.New("must be an HTTPS image.tmdb.org URL")
	}
	return nil
}

func nfoTarget(mediaPath string) (string, bool, error) {
	candidates := []string{strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".nfo", filepath.Join(filepath.Dir(mediaPath), "movie.nfo")}
	for _, candidate := range candidates {
		if _, err := os.Lstat(candidate); err == nil {
			return candidate, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("inspect NFO target: %w", err)
		}
	}
	return candidates[0], false, nil
}
