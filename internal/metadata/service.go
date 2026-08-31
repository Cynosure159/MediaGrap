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
	details, err := s.provider.Movie(ctx, providerID)
	if err != nil {
		return Record{}, err
	}
	record := Record{MediaItemID: itemID, Provider: "tmdb", ProviderID: details.ID, Title: details.Title, OriginalTitle: details.OriginalTitle, Year: details.Year, Overview: details.Overview, RuntimeMinutes: details.Runtime, Genres: details.Genres, PosterURL: details.PosterURL, BackdropURL: details.BackdropURL}
	return s.Save(ctx, record)
}
func (s *Service) Record(ctx context.Context, itemID int64) (Record, error) {
	var record Record
	var year, runtime sql.NullInt64
	var genres, locked string
	err := s.db.QueryRowContext(ctx, `SELECT media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,locked_fields_json,updated_at FROM media_metadata WHERE media_item_id=?`, itemID).Scan(&record.MediaItemID, &record.Provider, &record.ProviderID, &record.Title, &record.OriginalTitle, &year, &record.Overview, &runtime, &genres, &record.PosterURL, &record.BackdropURL, &locked, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return Record{MediaItemID: itemID, Genres: []string{}, LockedFields: []string{}}, nil
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
	_ = json.Unmarshal([]byte(genres), &record.Genres)
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
	return normalized(Record{MediaItemID: itemID, Provider: "tmdb", ProviderID: movie.TMDbID, Title: movie.Title, OriginalTitle: movie.OriginalTitle, Year: movie.Year, Overview: movie.Plot, RuntimeMinutes: movie.Runtime, Genres: movie.Genres, PosterURL: movie.PosterURL, BackdropURL: movie.BackdropURL, LockedFields: []string{}}), true, nil
}

func normalized(record Record) Record {
	if record.Genres == nil {
		record.Genres = []string{}
	}
	if record.LockedFields == nil {
		record.LockedFields = []string{}
	}
	return record
}
func (s *Service) Save(ctx context.Context, record Record) (Record, error) {
	if strings.TrimSpace(record.Title) == "" {
		return Record{}, errors.New("title is required")
	}
	genres, _ := json.Marshal(record.Genres)
	locked, _ := json.Marshal(record.LockedFields)
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_metadata(media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,locked_fields_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,datetime('now')) ON CONFLICT(media_item_id) DO UPDATE SET provider=excluded.provider,provider_id=excluded.provider_id,title=excluded.title,original_title=excluded.original_title,year=excluded.year,overview=excluded.overview,runtime_minutes=excluded.runtime_minutes,genres_json=excluded.genres_json,poster_url=excluded.poster_url,backdrop_url=excluded.backdrop_url,locked_fields_json=excluded.locked_fields_json,updated_at=datetime('now')`, record.MediaItemID, record.Provider, record.ProviderID, strings.TrimSpace(record.Title), strings.TrimSpace(record.OriginalTitle), record.Year, strings.TrimSpace(record.Overview), record.RuntimeMinutes, string(genres), record.PosterURL, record.BackdropURL, string(locked))
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
	content, err := kodi.WriteMovie(kodi.Movie{Title: record.Title, OriginalTitle: record.OriginalTitle, Year: record.Year, Plot: record.Overview, Runtime: record.RuntimeMinutes, Genres: record.Genres, TMDbID: record.ProviderID, PosterURL: record.PosterURL, BackdropURL: record.BackdropURL})
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
