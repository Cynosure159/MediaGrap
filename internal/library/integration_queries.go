package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// Integration DTOs deliberately exclude paths, raw payloads, messages and URLs.
type IntegrationMedia struct {
	ID       int64  `json:"id"`
	SourceID int64  `json:"sourceId"`
	Title    string `json:"title"`
	Year     *int   `json:"year"`
	Missing  bool   `json:"missing"`
}
type IntegrationJob struct {
	ID              int64  `json:"id"`
	SourceID        int64  `json:"sourceId"`
	Kind            string `json:"kind"`
	State           string `json:"state"`
	ProgressCurrent int    `json:"progressCurrent"`
	ProgressTotal   int    `json:"progressTotal"`
}

func (s *Service) IntegrationMedia(ctx context.Context, sources []int64, id, after int64, query string, limit int) ([]IntegrationMedia, error) {
	if limit < 1 || limit > 100 || len(query) > 256 {
		return nil, errors.New("invalid query")
	}
	ids, _ := json.Marshal(sources)
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.source_id,COALESCE(NULLIF(meta.title,''),m.title_hint),COALESCE(meta.year,m.year_hint),m.missing FROM media_items m LEFT JOIN media_metadata meta ON meta.media_item_id=m.id WHERE m.source_id IN (SELECT value FROM json_each(?)) AND (?=0 OR m.id=?) AND m.id>? AND (?='' OR instr(lower(COALESCE(NULLIF(meta.title,''),m.title_hint)),lower(?))>0) ORDER BY m.id LIMIT ?`, string(ids), id, id, after, query, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []IntegrationMedia{}
	for rows.Next() {
		var m IntegrationMedia
		if err = rows.Scan(&m.ID, &m.SourceID, &m.Title, &m.Year, &m.Missing); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}
func (s *Service) IntegrationJobs(ctx context.Context, sources []int64, id, after int64, limit int) ([]IntegrationJob, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("invalid query")
	}
	ids, _ := json.Marshal(sources)
	rows, err := s.db.QueryContext(ctx, `SELECT id,source_id,kind,state,progress_current,progress_total FROM jobs WHERE source_id IN (SELECT value FROM json_each(?)) AND (?=0 OR id=?) AND id>? ORDER BY id LIMIT ?`, string(ids), id, id, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []IntegrationJob{}
	for rows.Next() {
		var j IntegrationJob
		var source sql.NullInt64
		if err = rows.Scan(&j.ID, &source, &j.Kind, &j.State, &j.ProgressCurrent, &j.ProgressTotal); err != nil {
			return nil, err
		}
		if !source.Valid {
			continue
		}
		j.SourceID = source.Int64
		items = append(items, j)
	}
	return items, rows.Err()
}
