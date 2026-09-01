package library

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Repository interface {
	ListSources(ctx context.Context) ([]Source, error)
	GetSource(ctx context.Context, id int64) (Source, error)
	CreateSource(ctx context.Context, name, rootPath string) (Source, error)
	UpdateSource(ctx context.Context, id int64, name, rootPath string, enabled bool) (Source, error)
	DeleteSource(ctx context.Context, id int64) error
	ListMedia(ctx context.Context, query string, page, pageSize int) (Page, error)
	GetMediaItem(ctx context.Context, id int64) (MediaItem, error)
	GetMediaLocation(ctx context.Context, id int64) (MediaLocation, error)
	ListTVShows(ctx context.Context, query string) ([]TVShow, error)
	GetTVShowDetail(ctx context.Context, id int64) (TVShowDetail, error)
	GetSidecars(ctx context.Context, mediaItemID int64) ([]Sidecar, error)
}

type sqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id, s.name, s.root_path, s.enabled, COUNT(m.id) FROM sources s LEFT JOIN media_items m ON m.source_id = s.id AND m.missing = 0 GROUP BY s.id ORDER BY s.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Source, 0)
	for rows.Next() {
		var source Source
		var enabled int
		if err := rows.Scan(&source.ID, &source.Name, &source.RootPath, &enabled, &source.ItemCount); err != nil {
			return nil, err
		}
		source.Enabled = enabled == 1
		source.Writable = isWritable(source.RootPath)
		result = append(result, source)
	}
	return result, rows.Err()
}

func (r *sqliteRepository) GetSource(ctx context.Context, id int64) (Source, error) {
	var source Source
	var enabled int
	err := r.db.QueryRowContext(ctx, `SELECT s.id, s.name, s.root_path, s.enabled, COUNT(m.id) FROM sources s LEFT JOIN media_items m ON m.source_id = s.id AND m.missing = 0 WHERE s.id = ? GROUP BY s.id`, id).
		Scan(&source.ID, &source.Name, &source.RootPath, &enabled, &source.ItemCount)
	if err != nil {
		return Source{}, errors.New("source not found")
	}
	source.Enabled = enabled == 1
	source.Writable = isWritable(source.RootPath)
	return source, nil
}

func (r *sqliteRepository) CreateSource(ctx context.Context, name, rootPath string) (Source, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO sources(name, root_path, enabled) VALUES(?, ?, 1)`, strings.TrimSpace(name), strings.TrimSpace(rootPath))
	if err != nil {
		return Source{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Source{}, err
	}
	return r.GetSource(ctx, id)
}

func (r *sqliteRepository) UpdateSource(ctx context.Context, id int64, name, rootPath string, enabled bool) (Source, error) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	result, err := r.db.ExecContext(ctx, `UPDATE sources SET name=?, root_path=?, enabled=? WHERE id=?`, strings.TrimSpace(name), strings.TrimSpace(rootPath), enabledVal, id)
	if err != nil {
		return Source{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return Source{}, errors.New("source not found")
	}
	return r.GetSource(ctx, id)
}

func (r *sqliteRepository) DeleteSource(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sources WHERE id=?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return errors.New("source not found")
	}
	return nil
}

func (r *sqliteRepository) ListMedia(ctx context.Context, query string, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	trimmed := strings.TrimSpace(query)

	var countQuery string
	var selectQuery string
	var args []any

	if trimmed == "" {
		countQuery = `SELECT COUNT(*) FROM media_items m LEFT JOIN tv_episodes e ON e.media_item_id = m.id WHERE m.missing = 0 AND e.media_item_id IS NULL`
		selectQuery = `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, COALESFF_METADATA_TITLE(meta.title), COALESFF_METADATA_POSTER(meta.poster_url) FROM media_items m LEFT JOIN tv_episodes e ON e.media_item_id = m.id LEFT JOIN media_metadata meta ON meta.media_item_id = m.id WHERE m.missing = 0 AND e.media_item_id IS NULL ORDER BY m.relative_path COLLATE NOCASE LIMIT ? OFFSET ?`
		// Using standard SQL without custom UDFs:
		selectQuery = `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, COALESCE(meta.title, ''), COALESCE(meta.poster_url, '') FROM media_items m LEFT JOIN tv_episodes e ON e.media_item_id = m.id LEFT JOIN media_metadata meta ON meta.media_item_id = m.id WHERE m.missing = 0 AND e.media_item_id IS NULL ORDER BY m.relative_path COLLATE NOCASE LIMIT ? OFFSET ?`
		args = []any{pageSize, offset}
	} else {
		pattern := "%" + trimmed + "%"
		countQuery = `SELECT COUNT(*) FROM media_items m LEFT JOIN tv_episodes e ON e.media_item_id = m.id LEFT JOIN media_metadata meta ON meta.media_item_id = m.id WHERE m.missing = 0 AND e.media_item_id IS NULL AND (m.relative_path LIKE ? OR m.title_hint LIKE ? OR meta.title LIKE ?)`
		selectQuery = `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, COALESCE(meta.title, ''), COALESCE(meta.poster_url, '') FROM media_items m LEFT JOIN tv_episodes e ON e.media_item_id = m.id LEFT JOIN media_metadata meta ON meta.media_item_id = m.id WHERE m.missing = 0 AND e.media_item_id IS NULL AND (m.relative_path LIKE ? OR m.title_hint LIKE ? OR meta.title LIKE ?) ORDER BY m.relative_path COLLATE NOCASE LIMIT ? OFFSET ?`
		args = []any{pattern, pattern, pattern, pageSize, offset}
	}

	var total int
	countArgs := args
	if trimmed != "" {
		countArgs = []any{"%" + trimmed + "%", "%" + trimmed + "%", "%" + trimmed + "%"}
	} else {
		countArgs = nil
	}
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return Page{}, err
	}

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()

	items := make([]MediaItem, 0)
	for rows.Next() {
		var item MediaItem
		var year sql.NullInt64
		var title, poster string
		if err := rows.Scan(&item.ID, &item.SourceID, &item.RelativePath, &item.TitleHint, &year, &item.FileSize, &item.ModifiedAt, &title, &poster); err != nil {
			return Page{}, err
		}
		if year.Valid {
			y := int(year.Int64)
			item.YearHint = &y
		}
		item.Title = title
		item.PosterURL = poster
		items = append(items, item)
	}

	for i := range items {
		sidecars, err := r.GetSidecars(ctx, items[i].ID)
		if err == nil {
			items[i].Sidecars = sidecars
		}
	}

	return Page{Items: items, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

func (r *sqliteRepository) GetMediaItem(ctx context.Context, id int64) (MediaItem, error) {
	var item MediaItem
	var year sql.NullInt64
	var title, poster string
	err := r.db.QueryRowContext(ctx, `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, COALESCE(meta.title, ''), COALESCE(meta.poster_url, '') FROM media_items m LEFT JOIN media_metadata meta ON meta.media_item_id = m.id WHERE m.id = ? AND m.missing = 0`, id).
		Scan(&item.ID, &item.SourceID, &item.RelativePath, &item.TitleHint, &year, &item.FileSize, &item.ModifiedAt, &title, &poster)
	if err != nil {
		return MediaItem{}, errors.New("media item not found")
	}
	if year.Valid {
		y := int(year.Int64)
		item.YearHint = &y
	}
	item.Title = title
	item.PosterURL = poster
	sidecars, err := r.GetSidecars(ctx, id)
	if err == nil {
		item.Sidecars = sidecars
	}
	return item, nil
}

func (r *sqliteRepository) GetMediaLocation(ctx context.Context, id int64) (MediaLocation, error) {
	item, err := r.GetMediaItem(ctx, id)
	if err != nil {
		return MediaLocation{}, err
	}
	var root string
	err = r.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id = ?`, item.SourceID).Scan(&root)
	if err != nil {
		return MediaLocation{}, errors.New("media source not found")
	}
	fullPath := filepath.Join(root, item.RelativePath)
	return MediaLocation{
		Item:         item,
		AbsolutePath: fullPath,
		Writable:     isWritable(root),
	}, nil
}

func (r *sqliteRepository) ListTVShows(ctx context.Context, query string) ([]TVShow, error) {
	trimmed := strings.TrimSpace(query)
	var selectQuery string
	var args []any
	if trimmed == "" {
		selectQuery = `SELECT s.id, s.source_id, s.relative_path, s.title_hint, s.year_hint, (SELECT COUNT(DISTINCT e.season_number) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), (SELECT COUNT(e.id) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), COALESCE(meta.poster_url, '') FROM tv_shows s LEFT JOIN tv_metadata meta ON meta.show_id = s.id ORDER BY s.title_hint COLLATE NOCASE`
	} else {
		pattern := "%" + trimmed + "%"
		selectQuery = `SELECT s.id, s.source_id, s.relative_path, s.title_hint, s.year_hint, (SELECT COUNT(DISTINCT e.season_number) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), (SELECT COUNT(e.id) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), COALESCE(meta.poster_url, '') FROM tv_shows s LEFT JOIN tv_metadata meta ON meta.show_id = s.id WHERE s.title_hint LIKE ? OR s.relative_path LIKE ? OR meta.title LIKE ? ORDER BY s.title_hint COLLATE NOCASE`
		args = []any{pattern, pattern, pattern}
	}
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TVShow, 0)
	for rows.Next() {
		var show TVShow
		var year sql.NullInt64
		if err := rows.Scan(&show.ID, &show.SourceID, &show.RelativePath, &show.TitleHint, &year, &show.SeasonCount, &show.EpisodeCount, &show.PosterURL); err != nil {
			return nil, err
		}
		if year.Valid {
			y := int(year.Int64)
			show.YearHint = &y
		}
		items = append(items, show)
	}
	return items, rows.Err()
}

func (r *sqliteRepository) GetTVShowDetail(ctx context.Context, id int64) (TVShowDetail, error) {
	var show TVShow
	var year sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT s.id, s.source_id, s.relative_path, s.title_hint, s.year_hint, (SELECT COUNT(DISTINCT e.season_number) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), (SELECT COUNT(e.id) FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = s.id AND m.missing = 0), COALESCE(meta.poster_url, '') FROM tv_shows s LEFT JOIN tv_metadata meta ON meta.show_id = s.id WHERE s.id = ?`, id).
		Scan(&show.ID, &show.SourceID, &show.RelativePath, &show.TitleHint, &year, &show.SeasonCount, &show.EpisodeCount, &show.PosterURL)
	if err != nil {
		return TVShowDetail{}, errors.New("tv show not found")
	}
	if year.Valid {
		y := int(year.Int64)
		show.YearHint = &y
	}

	episodesRows, err := r.db.QueryContext(ctx, `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, e.season_number, e.episode_start, e.episode_end FROM tv_episodes e JOIN media_items m ON m.id = e.media_item_id WHERE e.show_id = ? AND m.missing = 0 ORDER BY e.season_number, e.episode_start`, id)
	if err != nil {
		return TVShowDetail{}, err
	}
	defer episodesRows.Close()

	episodes := make([]TVEpisode, 0)
	for episodesRows.Next() {
		var ep TVEpisode
		var epYear sql.NullInt64
		if err := episodesRows.Scan(&ep.ID, &ep.SourceID, &ep.RelativePath, &ep.TitleHint, &epYear, &ep.FileSize, &ep.ModifiedAt, &ep.SeasonNumber, &ep.EpisodeStart, &ep.EpisodeEnd); err != nil {
			return TVShowDetail{}, err
		}
		if epYear.Valid {
			y := int(epYear.Int64)
			ep.YearHint = &y
		}
		episodes = append(episodes, ep)
	}

	for i := range episodes {
		sidecars, err := r.GetSidecars(ctx, episodes[i].ID)
		if err == nil {
			episodes[i].Sidecars = sidecars
		}
	}

	var root string
	_ = r.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id = ?`, show.SourceID).Scan(&root)
	detail := TVShowDetail{
		Show:     show,
		Episodes: episodes,
		Writable: isWritable(root),
	}
	detail.Artwork = collectTVArtwork(root, detail)
	return detail, nil
}

func collectTVArtwork(root string, detail TVShowDetail) []TVArtwork {
	if root == "" {
		return []TVArtwork{}
	}
	directories := map[string]struct{}{filepath.Join(root, detail.Show.RelativePath): {}}
	for _, episode := range detail.Episodes {
		directories[filepath.Dir(filepath.Join(root, episode.RelativePath))] = struct{}{}
	}
	assets := []TVArtwork{}
	seen := make(map[string]struct{})
	for directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if sidecarExtensions[strings.ToLower(filepath.Ext(entry.Name()))] != "image" {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			info, statErr := entry.Info()
			if statErr != nil || !info.Mode().IsRegular() {
				continue
			}
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				continue
			}
			if _, ok := seen[relative]; ok {
				continue
			}
			seen[relative] = struct{}{}
			kind := "poster"
			lower := strings.ToLower(entry.Name())
			if strings.Contains(lower, "fanart") || strings.Contains(lower, "backdrop") || strings.Contains(lower, "background") {
				kind = "fanart"
			} else if strings.Contains(lower, "banner") {
				kind = "banner"
			} else if strings.Contains(lower, "thumb") || strings.Contains(lower, "landscape") {
				kind = "thumb"
			}
			assets = append(assets, TVArtwork{ID: relative, Kind: kind, RelativePath: relative})
		}
	}
	return assets
}

func (r *sqliteRepository) GetSidecars(ctx context.Context, mediaItemID int64) ([]Sidecar, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT relative_path, kind FROM sidecar_assets WHERE media_item_id = ? ORDER BY relative_path`, mediaItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Sidecar, 0)
	for rows.Next() {
		var s Sidecar
		if err := rows.Scan(&s.RelativePath, &s.Kind); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}
