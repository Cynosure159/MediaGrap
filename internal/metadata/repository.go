package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/mediagrap/mediagrap/internal/files"
)

type Repository interface {
	GetRecord(ctx context.Context, itemID int64) (Record, error)
	SaveRecord(ctx context.Context, record Record) error
	GetTVRecord(ctx context.Context, showID int64) (TVRecord, error)
	SaveTVRecord(ctx context.Context, record TVRecord) error
	GetWritePlan(ctx context.Context, id string) (WritePlan, error)
	SaveWritePlan(ctx context.Context, plan WritePlan) error
	UpdateWritePlanState(ctx context.Context, id, state string) error
	GetArtworkPlan(ctx context.Context, id string) (ArtworkPlan, error)
	SaveArtworkPlan(ctx context.Context, plan ArtworkPlan) error
	UpdateArtworkPlanState(ctx context.Context, id, state string) error
	InsertAuditEntry(ctx context.Context, action string, mediaItemID int64, targetPath, detail string) error
}

type sqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) GetRecord(ctx context.Context, itemID int64) (Record, error) {
	var record Record
	var year, runtime sql.NullInt64
	var genres, directors, writers, studios, cast, locked string
	var rating sql.NullFloat64
	var votes sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,rating,votes,content_rating,directors_json,writers_json,studios_json,cast_json,locked_fields_json,updated_at FROM media_metadata WHERE media_item_id=?`, itemID).
		Scan(&record.MediaItemID, &record.Provider, &record.ProviderID, &record.Title, &record.OriginalTitle, &year, &record.Overview, &runtime, &genres, &record.PosterURL, &record.BackdropURL, &rating, &votes, &record.ContentRating, &directors, &writers, &studios, &cast, &locked, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return normalized(Record{MediaItemID: itemID, Genres: []string{}, LockedFields: []string{}}), nil
	}
	if err != nil {
		return Record{}, err
	}
	if year.Valid {
		v := int(year.Int64)
		record.Year = &v
	}
	if runtime.Valid {
		v := int(runtime.Int64)
		record.RuntimeMinutes = &v
	}
	if rating.Valid {
		record.Rating = &rating.Float64
	}
	if votes.Valid {
		v := int(votes.Int64)
		record.Votes = &v
	}
	_ = json.Unmarshal([]byte(genres), &record.Genres)
	_ = json.Unmarshal([]byte(directors), &record.Directors)
	_ = json.Unmarshal([]byte(writers), &record.Writers)
	_ = json.Unmarshal([]byte(studios), &record.Studios)
	_ = json.Unmarshal([]byte(cast), &record.Cast)
	_ = json.Unmarshal([]byte(locked), &record.LockedFields)
	return normalized(record), nil
}

func (r *sqliteRepository) SaveRecord(ctx context.Context, record Record) error {
	record = normalized(record)
	genres, _ := json.Marshal(record.Genres)
	directors, _ := json.Marshal(record.Directors)
	writers, _ := json.Marshal(record.Writers)
	studios, _ := json.Marshal(record.Studios)
	cast, _ := json.Marshal(record.Cast)
	locked, _ := json.Marshal(record.LockedFields)
	_, err := r.db.ExecContext(ctx, `INSERT INTO media_metadata(media_item_id,provider,provider_id,title,original_title,year,overview,runtime_minutes,genres_json,poster_url,backdrop_url,rating,votes,content_rating,directors_json,writers_json,studios_json,cast_json,locked_fields_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now')) ON CONFLICT(media_item_id) DO UPDATE SET provider=excluded.provider,provider_id=excluded.provider_id,title=excluded.title,original_title=excluded.original_title,year=excluded.year,overview=excluded.overview,runtime_minutes=excluded.runtime_minutes,genres_json=excluded.genres_json,poster_url=excluded.poster_url,backdrop_url=excluded.backdrop_url,rating=excluded.rating,votes=excluded.votes,content_rating=excluded.content_rating,directors_json=excluded.directors_json,writers_json=excluded.writers_json,studios_json=excluded.studios_json,cast_json=excluded.cast_json,locked_fields_json=excluded.locked_fields_json,updated_at=datetime('now')`, record.MediaItemID, record.Provider, record.ProviderID, strings.TrimSpace(record.Title), strings.TrimSpace(record.OriginalTitle), record.Year, strings.TrimSpace(record.Overview), record.RuntimeMinutes, string(genres), record.PosterURL, record.BackdropURL, record.Rating, record.Votes, strings.TrimSpace(record.ContentRating), string(directors), string(writers), string(studios), string(cast), string(locked))
	return err
}

func (r *sqliteRepository) GetTVRecord(ctx context.Context, showID int64) (TVRecord, error) {
	var record TVRecord
	var year, votes sql.NullInt64
	var rating sql.NullFloat64
	var genres, cast string
	err := r.db.QueryRowContext(ctx, `SELECT show_id,provider,provider_id,title,original_title,year,overview,genres_json,poster_url,backdrop_url,rating,votes,status,network,cast_json,updated_at FROM tv_metadata WHERE show_id=?`, showID).
		Scan(&record.ShowID, &record.Provider, &record.ProviderID, &record.Title, &record.OriginalTitle, &year, &record.Overview, &genres, &record.PosterURL, &record.BackdropURL, &rating, &votes, &record.Status, &record.Network, &cast, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return normalizedTVRecord(TVRecord{ShowID: showID}), nil
	}
	if err != nil {
		return TVRecord{}, err
	}
	if year.Valid {
		v := int(year.Int64)
		record.Year = &v
	}
	if votes.Valid {
		v := int(votes.Int64)
		record.Votes = &v
	}
	if rating.Valid {
		record.Rating = &rating.Float64
	}
	_ = json.Unmarshal([]byte(genres), &record.Genres)
	_ = json.Unmarshal([]byte(cast), &record.Cast)

	episodesRows, err := r.db.QueryContext(ctx, `SELECT season_number,episode_number,title,overview,air_date,runtime_minutes,still_url FROM tv_episode_metadata WHERE show_id=? ORDER BY season_number,episode_number`, showID)
	if err != nil {
		return TVRecord{}, err
	}
	defer episodesRows.Close()

	record.Episodes = []TVEpisodeDetails{}
	for episodesRows.Next() {
		var episode TVEpisodeDetails
		var runtime sql.NullInt64
		if err := episodesRows.Scan(&episode.SeasonNumber, &episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.AirDate, &runtime, &episode.StillURL); err != nil {
			return TVRecord{}, err
		}
		if runtime.Valid {
			v := int(runtime.Int64)
			episode.RuntimeMinutes = &v
		}
		record.Episodes = append(record.Episodes, episode)
	}
	return normalizedTVRecord(record), episodesRows.Err()
}

func (r *sqliteRepository) SaveTVRecord(ctx context.Context, record TVRecord) error {
	record = normalizedTVRecord(record)
	genres, _ := json.Marshal(record.Genres)
	cast, _ := json.Marshal(record.Cast)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO tv_metadata(show_id,provider,provider_id,title,original_title,year,overview,genres_json,poster_url,backdrop_url,rating,votes,status,network,cast_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now')) ON CONFLICT(show_id) DO UPDATE SET provider=excluded.provider,provider_id=excluded.provider_id,title=excluded.title,original_title=excluded.original_title,year=excluded.year,overview=excluded.overview,genres_json=excluded.genres_json,poster_url=excluded.poster_url,backdrop_url=excluded.backdrop_url,rating=excluded.rating,votes=excluded.votes,status=excluded.status,network=excluded.network,cast_json=excluded.cast_json,updated_at=datetime('now')`, record.ShowID, record.Provider, record.ProviderID, strings.TrimSpace(record.Title), strings.TrimSpace(record.OriginalTitle), record.Year, strings.TrimSpace(record.Overview), string(genres), record.PosterURL, record.BackdropURL, record.Rating, record.Votes, strings.TrimSpace(record.Status), strings.TrimSpace(record.Network), string(cast))
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM tv_episode_metadata WHERE show_id=?`, record.ShowID); err != nil {
		return err
	}

	for _, episode := range record.Episodes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO tv_episode_metadata(show_id,season_number,episode_number,title,overview,air_date,runtime_minutes,still_url) VALUES(?,?,?,?,?,?,?,?)`, record.ShowID, episode.SeasonNumber, episode.EpisodeNumber, strings.TrimSpace(episode.Title), strings.TrimSpace(episode.Overview), strings.TrimSpace(episode.AirDate), episode.RuntimeMinutes, episode.StillURL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) GetWritePlan(ctx context.Context, id string) (WritePlan, error) {
	var plan WritePlan
	err := r.db.QueryRowContext(ctx, `SELECT id,media_item_id,target_path,content,state,created_at FROM write_plans WHERE id=?`, id).
		Scan(&plan.ID, &plan.MediaItemID, &plan.TargetPath, &plan.Content, &plan.State, &plan.CreatedAt)
	if err != nil {
		return WritePlan{}, errors.New("write plan not found")
	}
	plan.WillReplace, plan.Conflict, _ = files.ValidateTarget(plan.TargetPath)
	return plan, nil
}

func (r *sqliteRepository) SaveWritePlan(ctx context.Context, plan WritePlan) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO write_plans(id,media_item_id,target_path,content,state) VALUES(?,?,?,?,?)`, plan.ID, plan.MediaItemID, plan.TargetPath, plan.Content, plan.State)
	return err
}

func (r *sqliteRepository) UpdateWritePlanState(ctx context.Context, id, state string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE write_plans SET state=?,applied_at=datetime('now') WHERE id=?`, state, id)
	return err
}

func (r *sqliteRepository) GetArtworkPlan(ctx context.Context, id string) (ArtworkPlan, error) {
	var plan ArtworkPlan
	err := r.db.QueryRowContext(ctx, `SELECT id,media_item_id,state,created_at FROM artwork_plans WHERE id=?`, id).
		Scan(&plan.ID, &plan.MediaItemID, &plan.State, &plan.CreatedAt)
	if err != nil {
		return ArtworkPlan{}, errors.New("artwork plan not found")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT kind,source_url,target_path FROM artwork_plan_assets WHERE artwork_plan_id=? ORDER BY id`, id)
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
		asset.WillReplace, asset.Conflict, _ = files.ValidateTarget(asset.TargetPath)
		plan.Assets = append(plan.Assets, asset)
	}
	return plan, rows.Err()
}

func (r *sqliteRepository) SaveArtworkPlan(ctx context.Context, plan ArtworkPlan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `INSERT INTO artwork_plans(id,media_item_id,state) VALUES(?,?,?)`, plan.ID, plan.MediaItemID, plan.State); err != nil {
		return err
	}
	for _, asset := range plan.Assets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO artwork_plan_assets(artwork_plan_id,kind,source_url,target_path) VALUES(?,?,?,?)`, plan.ID, asset.Kind, asset.SourceURL, asset.TargetPath); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) UpdateArtworkPlanState(ctx context.Context, id, state string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE artwork_plans SET state=?,applied_at=datetime('now') WHERE id=?`, state, id)
	return err
}

func (r *sqliteRepository) InsertAuditEntry(ctx context.Context, action string, mediaItemID int64, targetPath, detail string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail) VALUES(?,?,?,?)`, action, mediaItemID, targetPath, detail)
	return err
}

func normalizedTVRecord(record TVRecord) TVRecord {
	if record.Genres == nil {
		record.Genres = []string{}
	}
	if record.Cast == nil {
		record.Cast = []Person{}
	}
	if record.Episodes == nil {
		record.Episodes = []TVEpisodeDetails{}
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return record
}
