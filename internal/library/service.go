package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var videoExtensions = map[string]bool{".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true, ".webm": true}
var sidecarExtensions = map[string]string{".nfo": "nfo", ".jpg": "image", ".jpeg": "image", ".png": "image", ".webp": "image"}
var yearPattern = regexp.MustCompile(`(?i)[. _\-\[(](19\d{2}|20\d{2})[. _\-\])]`)

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
	FileSize     int64     `json:"fileSize"`
	ModifiedAt   string    `json:"modifiedAt"`
	Sidecars     []Sidecar `json:"sidecars"`
}
type Sidecar struct {
	RelativePath string `json:"relativePath"`
	Kind         string `json:"kind"`
}
type Job struct {
	ID              int64  `json:"id"`
	Kind            string `json:"kind"`
	SourceID        *int64 `json:"sourceId"`
	State           string `json:"state"`
	ProgressCurrent int    `json:"progressCurrent"`
	ProgressTotal   int    `json:"progressTotal"`
	Message         string `json:"message"`
	ErrorMessage    string `json:"errorMessage"`
	CreatedAt       string `json:"createdAt"`
}
type Page struct {
	Items    []MediaItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type Service struct {
	db    *sql.DB
	roots []string
	locks sync.Map
}

func NewService(db *sql.DB, roots []string) *Service { return &Service{db: db, roots: roots} }

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

func (s *Service) QueueScan(ctx context.Context, sourceID int64) (Job, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources WHERE id = ? AND enabled = 1`, sourceID).Scan(&exists); err != nil || exists == 0 {
		return Job{}, errors.New("source not found")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO jobs(kind, source_id, state, message) VALUES('scan', ?, 'queued', 'Waiting for a worker')`, sourceID)
	if err != nil {
		return Job{}, err
	}
	id, _ := result.LastInsertId()
	return s.Job(ctx, id)
}

func (s *Service) ListJobs(ctx context.Context) ([]Job, error) {
	return s.listJobs(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, created_at FROM jobs ORDER BY id DESC LIMIT 50`)
}
func (s *Service) Job(ctx context.Context, id int64) (Job, error) {
	jobs, err := s.listJobs(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, created_at FROM jobs WHERE id = `+fmt.Sprint(id))
	if err != nil || len(jobs) == 0 {
		return Job{}, errors.New("job not found")
	}
	return jobs[0], nil
}
func (s *Service) listJobs(ctx context.Context, query string) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []Job{}
	for rows.Next() {
		var job Job
		var source sql.NullInt64
		if err := rows.Scan(&job.ID, &job.Kind, &source, &job.State, &job.ProgressCurrent, &job.ProgressTotal, &job.Message, &job.ErrorMessage, &job.CreatedAt); err != nil {
			return nil, err
		}
		if source.Valid {
			job.SourceID = &source.Int64
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
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
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items WHERE missing = 0 AND title_hint LIKE ?`, "%"+query+"%").Scan(&total); err != nil {
		return Page{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, source_id, relative_path, title_hint, year_hint, file_size, modified_at FROM media_items WHERE missing = 0 AND title_hint LIKE ? ORDER BY title_hint COLLATE NOCASE LIMIT ? OFFSET ?`, "%"+query+"%", pageSize, (page-1)*pageSize)
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
		item.Sidecars, _ = s.sidecars(ctx, item.ID)
		items = append(items, item)
	}
	return Page{Items: items, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}
func (s *Service) Media(ctx context.Context, id int64) (MediaItem, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, source_id, relative_path, title_hint, year_hint, file_size, modified_at FROM media_items WHERE id=? AND missing=0`, id)
	item, err := scanItem(row)
	if err != nil {
		return MediaItem{}, errors.New("media item not found")
	}
	item.Sidecars, _ = s.sidecars(ctx, item.ID)
	return item, nil
}
func scanItem(row interface{ Scan(...any) error }) (MediaItem, error) {
	var item MediaItem
	var year sql.NullInt64
	err := row.Scan(&item.ID, &item.SourceID, &item.RelativePath, &item.TitleHint, &year, &item.FileSize, &item.ModifiedAt)
	if year.Valid {
		v := int(year.Int64)
		item.YearHint = &v
	}
	return item, err
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
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOne(ctx)
		}
	}
}
func (s *Service) runOne(ctx context.Context) {
	var id, sourceID int64
	err := s.db.QueryRowContext(ctx, `SELECT id,source_id FROM jobs WHERE state='queued' AND kind='scan' ORDER BY id LIMIT 1`).Scan(&id, &sourceID)
	if err != nil {
		return
	}
	result, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='running',started_at=datetime('now'),updated_at=datetime('now'),message='Scanning source' WHERE id=? AND state='queued'`, id)
	if err != nil {
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return
	}
	if err := s.scan(ctx, id, sourceID); err != nil {
		s.db.ExecContext(ctx, `UPDATE jobs SET state='failed',error_message=?,completed_at=datetime('now'),updated_at=datetime('now') WHERE id=?`, err.Error(), id)
		return
	}
	s.db.ExecContext(ctx, `UPDATE jobs SET state='succeeded',message='Scan complete',completed_at=datetime('now'),updated_at=datetime('now') WHERE id=?`, id)
}
func (s *Service) scan(ctx context.Context, jobID, sourceID int64) error {
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
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
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
		count++
		if count%25 == 0 {
			s.db.ExecContext(ctx, `UPDATE jobs SET progress_current=?,message='Indexed media files',updated_at=datetime('now') WHERE id=?`, count, jobID)
		}
		return nil
	})
}
func (s *Service) recordSidecars(ctx context.Context, itemID int64, root, relative string) {
	directory := filepath.Dir(filepath.Join(root, relative))
	base := strings.TrimSuffix(filepath.Base(relative), filepath.Ext(relative))
	matches, _ := filepath.Glob(filepath.Join(directory, base+".*"))
	s.db.ExecContext(ctx, `DELETE FROM sidecar_assets WHERE media_item_id=?`, itemID)
	for _, match := range matches {
		extension := strings.ToLower(filepath.Ext(match))
		kind, ok := sidecarExtensions[extension]
		if !ok {
			continue
		}
		rel, err := filepath.Rel(root, match)
		if err == nil {
			s.db.ExecContext(ctx, `INSERT OR IGNORE INTO sidecar_assets(media_item_id,relative_path,kind) VALUES(?,?,?)`, itemID, rel, kind)
		}
	}
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
func isWritable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0o200 != 0
}
