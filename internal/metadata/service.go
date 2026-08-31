package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
}
type Service struct {
	db       *sql.DB
	provider Provider
}

func NewService(db *sql.DB, provider Provider) *Service { return &Service{db: db, provider: provider} }

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
	target := strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".nfo"
	plan := WritePlan{ID: uuid.NewString(), MediaItemID: record.MediaItemID, TargetPath: target, Content: string(content), State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO write_plans(id,media_item_id,target_path,content,state) VALUES(?,?,?,?,?)`, plan.ID, plan.MediaItemID, plan.TargetPath, plan.Content, plan.State); err != nil {
		return WritePlan{}, err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('nfo.preview',?,?,?)`, plan.MediaItemID, plan.TargetPath, "NFO write plan created")
	return plan, nil
}
func (s *Service) Plan(ctx context.Context, id string) (WritePlan, error) {
	var plan WritePlan
	err := s.db.QueryRowContext(ctx, `SELECT id,media_item_id,target_path,content,state,created_at FROM write_plans WHERE id=?`, id).Scan(&plan.ID, &plan.MediaItemID, &plan.TargetPath, &plan.Content, &plan.State, &plan.CreatedAt)
	if err != nil {
		return WritePlan{}, errors.New("write plan not found")
	}
	_, err = os.Stat(plan.TargetPath)
	plan.Conflict = err == nil
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
		return WritePlan{}, errors.New("target NFO already exists; MediaGrap will not overwrite it")
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
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES('nfo.apply',?,?,?)`, plan.MediaItemID, plan.TargetPath, "NFO created atomically")
	return s.Plan(ctx, id)
}
