package library

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mediagrap/mediagrap/internal/jobs"
)

var videoExtensions = map[string]bool{".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true, ".webm": true}
var sidecarExtensions = map[string]string{".nfo": "nfo", ".jpg": "image", ".jpeg": "image", ".png": "image", ".webp": "image"}
var yearPattern = regexp.MustCompile(`(?i)[. _\-\[(](19\d{2}|20\d{2})[. _\-\])]`)
var seasonDirectoryPattern = regexp.MustCompile(`(?i)^(?:season[ ._-]*)?s?(\d{1,2})$`)
var episodePattern = regexp.MustCompile(`(?i)(?:^|[. _-])s(\d{1,2})e(\d{1,3})(?:e(\d{1,3}))?(?:$|[. _-])`)
var episodeXPattern = regexp.MustCompile(`(?i)(?:^|[. _-])(\d{1,2})x(\d{1,3})(?:-(\d{1,3}))?(?:$|[. _-])`)

type Source struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	RootPath  string `json:"rootPath"`
	Enabled   bool   `json:"enabled"`
	ItemCount int    `json:"itemCount"`
	Writable  bool   `json:"writable"`
}
type MediaItem struct {
	ID           int64     `json:"id"`
	SourceID     int64     `json:"sourceId"`
	RelativePath string    `json:"relativePath"`
	TitleHint    string    `json:"titleHint"`
	YearHint     *int      `json:"yearHint"`
	Title        string    `json:"title,omitempty"`
	PosterURL    string    `json:"posterUrl,omitempty"`
	FileSize     int64     `json:"fileSize"`
	ModifiedAt   string    `json:"modifiedAt"`
	Sidecars     []Sidecar `json:"sidecars"`
}
type MediaLocation struct {
	Item         MediaItem
	AbsolutePath string
	Writable     bool
}
type Sidecar struct {
	RelativePath string `json:"relativePath"`
	Kind         string `json:"kind"`
}
type Job = jobs.Job
type Page struct {
	Items    []MediaItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
type TVShow struct {
	ID           int64  `json:"id"`
	SourceID     int64  `json:"sourceId"`
	RelativePath string `json:"relativePath"`
	TitleHint    string `json:"titleHint"`
	YearHint     *int   `json:"yearHint"`
	EpisodeCount int    `json:"episodeCount"`
	SeasonCount  int    `json:"seasonCount"`
	PosterURL    string `json:"posterUrl,omitempty"`
}
type TVEpisode struct {
	MediaItem
	SeasonNumber int `json:"seasonNumber"`
	EpisodeStart int `json:"episodeStart"`
	EpisodeEnd   int `json:"episodeEnd"`
}
type TVShowDetail struct {
	Show     TVShow      `json:"show"`
	Episodes []TVEpisode `json:"episodes"`
	Artwork  []TVArtwork `json:"artwork"`
	Writable bool        `json:"writable"`
}
type TVArtwork struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relativePath"`
}

type Service struct {
	db               *sql.DB
	roots            []string
	locks            sync.Map
	logger           *slog.Logger
	jobs             *jobs.Service
	metadataHydrator interface {
		HydrateExistingNFO(context.Context, int64, string) error
	}
}

func NewService(db *sql.DB, roots []string) *Service {
	logger := slog.Default()
	s := &Service{db: db, roots: roots, logger: logger, jobs: jobs.NewService(db, logger)}
	s.registerScanJobHandler()
	return s
}

func (s *Service) SetLogger(logger *slog.Logger) {
	if logger != nil {
		s.logger = logger
		s.jobs = jobs.NewService(s.db, logger)
		s.registerScanJobHandler()
	}
}

func (s *Service) registerScanJobHandler() {
	s.jobs.RegisterHandler("scan", func(ctx context.Context, job jobs.Job, updateProgress func(current int, message string)) error {
		if job.SourceID == nil {
			return errors.New("source id is required for scan job")
		}
		return s.scan(ctx, job.ID, *job.SourceID, updateProgress)
	})
}

func (s *Service) SetMetadataHydrator(hydrator interface {
	HydrateExistingNFO(context.Context, int64, string) error
}) {
	s.metadataHydrator = hydrator
}

func (s *Service) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, s.root_path, s.enabled, COUNT(m.id) FROM sources s LEFT JOIN media_items m ON m.source_id = s.id AND m.missing = 0 GROUP BY s.id ORDER BY s.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Source{}
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

func (s *Service) CreateSource(ctx context.Context, name, rootPath string) (Source, error) {
	name = strings.TrimSpace(name)
	rootPath = filepath.Clean(strings.TrimSpace(rootPath))
	if len(name) < 2 || len(name) > 120 {
		return Source{}, errors.New("source name must be 2 to 120 characters")
	}
	if !s.allowed(rootPath) {
		return Source{}, errors.New("source path is outside configured media roots")
	}
	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		return Source{}, errors.New("source path must be an accessible directory")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO sources(name, root_path) VALUES(?, ?)`, name, rootPath)
	if err != nil {
		return Source{}, fmt.Errorf("create source: %w", err)
	}
	id, _ := result.LastInsertId()
	return Source{ID: id, Name: name, RootPath: rootPath, Enabled: true, Writable: isWritable(rootPath)}, nil
}

// DeleteSource removes only MediaGrap's source configuration and its indexed
// records. It never touches the mounted media files themselves.
func (s *Service) DeleteSource(ctx context.Context, sourceID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM sources WHERE id=?`, sourceID)
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete source result: %w", err)
	}
	if deleted == 0 {
		return errors.New("source not found")
	}
	s.logger.Info("media source deleted", "source_id", sourceID)
	return nil
}

func (s *Service) QueueScan(ctx context.Context, sourceID int64) (Job, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources WHERE id = ? AND enabled = 1`, sourceID).Scan(&exists); err != nil || exists == 0 {
		return Job{}, errors.New("source not found")
	}
	return s.jobs.Queue(ctx, "scan", &sourceID)
}

func (s *Service) ListJobs(ctx context.Context) ([]Job, error) {
	return s.jobs.List(ctx)
}

func (s *Service) Job(ctx context.Context, id int64) (Job, error) {
	return s.jobs.Get(ctx, id)
}

func (s *Service) ListMedia(ctx context.Context, query string, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}
	query = strings.TrimSpace(query)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items m LEFT JOIN media_metadata mm ON mm.media_item_id = m.id WHERE m.missing = 0 AND (m.title_hint LIKE ? OR mm.title LIKE ?) AND NOT EXISTS(SELECT 1 FROM tv_episodes e WHERE e.media_item_id=m.id)`, "%"+query+"%", "%"+query+"%").Scan(&total); err != nil {
		return Page{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, mm.title, mm.poster_url FROM media_items m LEFT JOIN media_metadata mm ON mm.media_item_id = m.id WHERE m.missing = 0 AND (m.title_hint LIKE ? OR mm.title LIKE ?) AND NOT EXISTS(SELECT 1 FROM tv_episodes e WHERE e.media_item_id=m.id) ORDER BY COALESCE(NULLIF(mm.title, ''), m.title_hint) COLLATE NOCASE LIMIT ? OFFSET ?`, "%"+query+"%", "%"+query+"%", pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	items := []MediaItem{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return Page{}, err
		}
		item.Sidecars = currentSidecarsForMedia(s.sourceRoot(ctx, item.SourceID), item.RelativePath)
		items = append(items, item)
	}
	return Page{Items: items, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

func (s *Service) ListTVShows(ctx context.Context, query string) ([]TVShow, error) {
	query = strings.TrimSpace(query)
	rows, err := s.db.QueryContext(ctx, `SELECT sh.id, sh.source_id, sh.relative_path, sh.title_hint, sh.year_hint, COUNT(DISTINCT e.media_item_id), COUNT(DISTINCT e.season_number)
		FROM tv_shows sh JOIN tv_episodes e ON e.show_id=sh.id JOIN media_items m ON m.id=e.media_item_id
		WHERE m.missing=0 AND sh.title_hint LIKE ? GROUP BY sh.id ORDER BY sh.title_hint COLLATE NOCASE`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	shows := []TVShow{}
	for rows.Next() {
		var show TVShow
		var year sql.NullInt64
		if err := rows.Scan(&show.ID, &show.SourceID, &show.RelativePath, &show.TitleHint, &year, &show.EpisodeCount, &show.SeasonCount); err != nil {
			return nil, err
		}
		if year.Valid {
			value := int(year.Int64)
			show.YearHint = &value
		}
		shows = append(shows, show)
	}
	return shows, rows.Err()
}

func (s *Service) TVShow(ctx context.Context, id int64) (TVShowDetail, error) {
	shows, err := s.ListTVShows(ctx, "")
	if err != nil {
		return TVShowDetail{}, err
	}
	var show *TVShow
	for index := range shows {
		if shows[index].ID == id {
			show = &shows[index]
			break
		}
	}
	if show == nil {
		return TVShowDetail{}, errors.New("TV show not found")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.source_id,m.relative_path,m.title_hint,m.year_hint,m.file_size,m.modified_at,e.season_number,e.episode_start,e.episode_end FROM tv_episodes e JOIN media_items m ON m.id=e.media_item_id WHERE e.show_id=? AND m.missing=0 ORDER BY e.season_number,e.episode_start,m.relative_path`, id)
	if err != nil {
		return TVShowDetail{}, err
	}
	defer rows.Close()
	root := s.sourceRoot(ctx, show.SourceID)
	detail := TVShowDetail{Show: *show, Writable: s.sourceWritable(ctx, show.SourceID), Episodes: []TVEpisode{}, Artwork: []TVArtwork{}}
	for rows.Next() {
		var episode TVEpisode
		var year sql.NullInt64
		if err := rows.Scan(&episode.ID, &episode.SourceID, &episode.RelativePath, &episode.TitleHint, &year, &episode.FileSize, &episode.ModifiedAt, &episode.SeasonNumber, &episode.EpisodeStart, &episode.EpisodeEnd); err != nil {
			return TVShowDetail{}, err
		}
		if year.Valid {
			value := int(year.Int64)
			episode.YearHint = &value
		}
		episode.Sidecars = currentSidecarsForMedia(root, episode.RelativePath)
		detail.Episodes = append(detail.Episodes, episode)
	}
	if err := rows.Err(); err != nil {
		return TVShowDetail{}, err
	}
	detail.Artwork = s.tvArtwork(ctx, detail)
	return detail, nil
}

// currentSidecarsForMedia is the one discovery rule used both by scans and by
// read APIs. Besides same-basename files, a directory with one video owns the
// standard Kodi directory-level assets such as movie.nfo and poster.jpg.
func currentSidecarsForMedia(root, relative string) []Sidecar {
	if root == "" {
		return []Sidecar{}
	}
	directory := filepath.Dir(filepath.Join(root, relative))
	base := strings.TrimSuffix(filepath.Base(relative), filepath.Ext(relative))
	matches, _ := filepath.Glob(filepath.Join(directory, base+".*"))
	if directoryHasOneVideo(directory) {
		entries, _ := os.ReadDir(directory)
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if _, ok := sidecarExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; ok {
				matches = append(matches, filepath.Join(directory, entry.Name()))
			}
		}
	}
	assets := []Sidecar{}
	seen := make(map[string]struct{})
	for _, match := range matches {
		if _, exists := seen[match]; exists {
			continue
		}
		seen[match] = struct{}{}
		extension := strings.ToLower(filepath.Ext(match))
		kind, ok := sidecarExtensions[extension]
		if !ok {
			continue
		}
		info, err := os.Lstat(match)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			continue
		}
		path, err := filepath.Rel(root, match)
		if err == nil {
			assets = append(assets, Sidecar{RelativePath: path, Kind: kind})
		}
	}
	return assets
}

func directoryHasOneVideo(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !videoExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		count++
	}
	return count == 1
}

func (s *Service) tvArtwork(ctx context.Context, detail TVShowDetail) []TVArtwork {
	var root string
	if err := s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, detail.Show.SourceID).Scan(&root); err != nil {
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
			if relErr != nil || !s.allowed(path) {
				continue
			}
			if _, ok := seen[relative]; ok {
				continue
			}
			seen[relative] = struct{}{}
			name := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			kind := "image"
			if name == "poster" || name == "folder" || name == "cover" {
				kind = "poster"
			} else if name == "fanart" || name == "backdrop" || name == "landscape" {
				kind = "fanart"
			}
			assets = append(assets, TVArtwork{ID: base64.RawURLEncoding.EncodeToString([]byte(relative)), Kind: kind, RelativePath: relative})
		}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].RelativePath < assets[j].RelativePath })
	return assets
}
func (s *Service) Media(ctx context.Context, id int64) (MediaItem, error) {
	row := s.db.QueryRowContext(ctx, `SELECT m.id, m.source_id, m.relative_path, m.title_hint, m.year_hint, m.file_size, m.modified_at, mm.title, mm.poster_url FROM media_items m LEFT JOIN media_metadata mm ON mm.media_item_id=m.id WHERE m.id=? AND m.missing=0`, id)
	item, err := scanItem(row)
	if err != nil {
		return MediaItem{}, errors.New("media item not found")
	}
	item.Sidecars = currentSidecarsForMedia(s.sourceRoot(ctx, item.SourceID), item.RelativePath)
	return item, nil
}
func (s *Service) LocateMedia(ctx context.Context, id int64) (MediaLocation, error) {
	item, err := s.Media(ctx, id)
	if err != nil {
		return MediaLocation{}, err
	}
	var root string
	if err := s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, item.SourceID).Scan(&root); err != nil {
		return MediaLocation{}, errors.New("media source not found")
	}
	path := filepath.Join(root, item.RelativePath)
	if !s.allowed(path) {
		return MediaLocation{}, errors.New("media path is outside configured roots")
	}
	return MediaLocation{Item: item, AbsolutePath: path, Writable: isWritable(root)}, nil
}

func (s *Service) Allowed(path string) bool { return s.allowed(path) }

func (s *Service) sourceRoot(ctx context.Context, sourceID int64) string {
	var root string
	_ = s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, sourceID).Scan(&root)
	return root
}

// MetadataSearchHint prefers the movie directory name because it commonly
// contains the release title while the filename often includes codec and group tags.
// Files directly in a source root have no directory hint and use the file-derived one.
func (s *Service) MetadataSearchHint(item MediaItem) (string, *int, string) {
	directory := filepath.Dir(item.RelativePath)
	if directory != "." {
		if title, year := parseHint(filepath.Base(directory)); title != "" {
			return title, year, "parent_directory"
		}
	}
	return item.TitleHint, item.YearHint, "filename"
}

func scanItem(row interface{ Scan(...any) error }) (MediaItem, error) {
	var item MediaItem
	var year sql.NullInt64
	var title, posterURL sql.NullString
	err := row.Scan(&item.ID, &item.SourceID, &item.RelativePath, &item.TitleHint, &year, &item.FileSize, &item.ModifiedAt, &title, &posterURL)
	if err != nil {
		return MediaItem{}, err
	}
	if year.Valid {
		v := int(year.Int64)
		item.YearHint = &v
	}
	if title.Valid {
		item.Title = title.String
	}
	if posterURL.Valid {
		item.PosterURL = posterURL.String
	}
	return item, nil
}
func (s *Service) sidecars(ctx context.Context, id int64) ([]Sidecar, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT relative_path,kind FROM sidecar_assets WHERE media_item_id=? ORDER BY relative_path`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Sidecar{}
	for rows.Next() {
		var a Sidecar
		if err := rows.Scan(&a.RelativePath, &a.Kind); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *Service) RunWorker(ctx context.Context) {
	s.jobs.RunWorker(ctx)
}

func (s *Service) scan(ctx context.Context, jobID, sourceID int64, progress func(current int, message string)) error {
	var root string
	if err := s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, sourceID).Scan(&root); err != nil {
		return err
	}
	lockValue, _ := s.locks.LoadOrStore(sourceID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if _, err := s.db.ExecContext(ctx, `UPDATE media_items SET missing=1 WHERE source_id=?`, sourceID); err != nil {
		return err
	}
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			if path != root && entry.Type()&fs.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !videoExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		title, year := parseHint(filepath.Base(path))
		var itemID int64
		err = s.db.QueryRowContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,year_hint,file_size,modified_at,missing,last_seen_at) VALUES(?,?,?,?,?,?,0,datetime('now')) ON CONFLICT(source_id,relative_path) DO UPDATE SET title_hint=excluded.title_hint,year_hint=excluded.year_hint,file_size=excluded.file_size,modified_at=excluded.modified_at,missing=0,last_seen_at=datetime('now') RETURNING id`, sourceID, relative, title, year, info.Size(), info.ModTime().UTC().Format(time.RFC3339)).Scan(&itemID)
		if err != nil {
			return err
		}
		s.recordSidecars(ctx, itemID, root, relative)
		s.recordTVEpisode(ctx, itemID, sourceID, relative)
		if s.metadataHydrator != nil {
			if _, _, _, isEpisode := parseEpisodeHint(filepath.Base(relative)); !isEpisode {
				if hydrateErr := s.metadataHydrator.HydrateExistingNFO(ctx, itemID, path); hydrateErr != nil {
					s.logger.Warn("existing movie NFO hydration failed", "media_item_id", itemID, "error", hydrateErr)
				}
			}
		}
		count++
		if count%25 == 0 && progress != nil {
			progress(count, "Indexed media files")
		}
		return nil
	})
	if err == nil {
		s.logger.Info("library scan indexed files", "job_id", jobID, "source_id", sourceID, "media_item_count", count)
	}
	return err
}
func (s *Service) recordSidecars(ctx context.Context, itemID int64, root, relative string) {
	s.db.ExecContext(ctx, `DELETE FROM sidecar_assets WHERE media_item_id=?`, itemID)
	for _, asset := range currentSidecarsForMedia(root, relative) {
		s.db.ExecContext(ctx, `INSERT OR IGNORE INTO sidecar_assets(media_item_id,relative_path,kind) VALUES(?,?,?)`, itemID, asset.RelativePath, asset.Kind)
	}
}

func (s *Service) recordTVEpisode(ctx context.Context, itemID, sourceID int64, relative string) {
	season, firstEpisode, lastEpisode, ok := parseEpisodeHint(filepath.Base(relative))
	if !ok {
		s.db.ExecContext(ctx, `DELETE FROM tv_episodes WHERE media_item_id=?`, itemID)
		return
	}
	showPath, showTitle, showYear := tvShowHint(relative)
	if showTitle == "" {
		return
	}
	var showID int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO tv_shows(source_id,relative_path,title_hint,year_hint,updated_at) VALUES(?,?,?,?,datetime('now')) ON CONFLICT(source_id,relative_path) DO UPDATE SET title_hint=excluded.title_hint,year_hint=excluded.year_hint,updated_at=datetime('now') RETURNING id`, sourceID, showPath, showTitle, showYear).Scan(&showID)
	if err != nil {
		return
	}
	var seasonID int64
	err = s.db.QueryRowContext(ctx, `INSERT INTO tv_seasons(show_id,season_number) VALUES(?,?) ON CONFLICT(show_id,season_number) DO UPDATE SET season_number=excluded.season_number RETURNING id`, showID, season).Scan(&seasonID)
	if err != nil {
		return
	}
	title := episodeTitleHint(filepath.Base(relative))
	s.db.ExecContext(ctx, `INSERT INTO tv_episodes(media_item_id,show_id,season_id,season_number,episode_start,episode_end,title_hint) VALUES(?,?,?,?,?,?,?) ON CONFLICT(media_item_id) DO UPDATE SET show_id=excluded.show_id,season_id=excluded.season_id,season_number=excluded.season_number,episode_start=excluded.episode_start,episode_end=excluded.episode_end,title_hint=excluded.title_hint`, itemID, showID, seasonID, season, firstEpisode, lastEpisode, title)
}

func (s *Service) sourceWritable(ctx context.Context, sourceID int64) bool {
	var root string
	if s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, sourceID).Scan(&root) != nil {
		return false
	}
	return isWritable(root)
}
func (s *Service) allowed(path string) bool {
	for _, root := range s.roots {
		relative, err := filepath.Rel(root, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
func parseHint(filename string) (string, *int) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	yearMatch := yearPattern.FindStringSubmatch(" " + name)
	var year *int
	if len(yearMatch) > 1 {
		value := 0
		fmt.Sscanf(yearMatch[1], "%d", &value)
		year = &value
		name = strings.Replace(name, yearMatch[0], " ", 1)
	}
	title := strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(name)
	title = strings.Join(strings.Fields(title), " ")
	return title, year
}

func parseEpisodeHint(filename string) (season, firstEpisode, lastEpisode int, ok bool) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	match := episodePattern.FindStringSubmatch(name)
	if len(match) == 0 {
		match = episodeXPattern.FindStringSubmatch(name)
	}
	if len(match) == 0 {
		return 0, 0, 0, false
	}
	fmt.Sscanf(match[1], "%d", &season)
	fmt.Sscanf(match[2], "%d", &firstEpisode)
	lastEpisode = firstEpisode
	if len(match) > 3 && match[3] != "" {
		fmt.Sscanf(match[3], "%d", &lastEpisode)
	}
	return season, firstEpisode, lastEpisode, season >= 0 && firstEpisode >= 0 && lastEpisode >= firstEpisode
}

func tvShowHint(relative string) (string, string, *int) {
	directory := filepath.Dir(relative)
	showDirectory := directory
	if seasonDirectoryPattern.MatchString(filepath.Base(directory)) {
		showDirectory = filepath.Dir(directory)
	}
	if showDirectory != "." {
		title, year := parseHint(filepath.Base(showDirectory))
		return showDirectory, title, year
	}
	filename := filepath.Base(relative)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	match := episodePattern.FindStringIndex(name)
	if match == nil {
		match = episodeXPattern.FindStringIndex(name)
	}
	if match == nil {
		return ".", "", nil
	}
	title, year := parseHint(name[:match[0]])
	return ".", title, year
}

func episodeTitleHint(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	if match := episodePattern.FindStringIndex(name); match != nil {
		return strings.Trim(strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(name[match[1]:]), " ")
	}
	if match := episodeXPattern.FindStringIndex(name); match != nil {
		return strings.Trim(strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(name[match[1]:]), " ")
	}
	return ""
}
func isWritable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0o200 != 0
}
