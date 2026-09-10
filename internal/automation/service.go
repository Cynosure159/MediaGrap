// Package automation coordinates authorized durable work and immutable plans.
package automation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/mediagrap/mediagrap/internal/jobs"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/tokens"
)

var ErrDenied = errors.New("NOT_FOUND_OR_FORBIDDEN")
var ErrConflict = errors.New("REQUEST_CONFLICT")
var ErrInvalid = errors.New("INVALID_ARGUMENT")

type Service struct {
	backlogLimit int
	linkSource   func(*os.Root, string, string) error
	db           *sql.DB
	tokens       *tokens.Service
	library      *library.Service
	metadata     *metadata.Service
}

func New(db *sql.DB, ts *tokens.Service, lib *library.Service, meta *metadata.Service) *Service {
	s := &Service{db: db, tokens: ts, library: lib, metadata: meta, backlogLimit: 100000, linkSource: func(root *os.Root, source, target string) error { return root.Link(source, target) }}
	lib.RegisterJobHandler("automation_task", s.runTask)
	lib.RegisterJobHandler("automation_apply", s.runPlan)
	return s
}

type TaskInput struct {
	SourceID       int64  `json:"sourceId,omitempty"`
	MediaID        int64  `json:"mediaId,omitempty"`
	ShowID         int64  `json:"showId,omitempty"`
	Season         int    `json:"season,omitempty"`
	Episode        *int   `json:"episode,omitempty"`
	Query          string `json:"query,omitempty"`
	IdempotencyKey string `json:"idempotencyKey"`
}
type Queued struct {
	JobID int64  `json:"jobId"`
	State string `json:"state"`
}
type taskPayload struct {
	TokenID string    `json:"tokenId"`
	Tool    string    `json:"tool"`
	Input   TaskInput `json:"input"`
}

func hash(value any) string {
	body, _ := json.Marshal(value)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
func (s *Service) source(ctx context.Context, p tokens.Token, in TaskInput) (int64, error) {
	id := in.SourceID
	if in.MediaID > 0 {
		items, err := s.library.IntegrationMedia(ctx, p.SourceIDs, in.MediaID, 0, "", 1)
		if err != nil || len(items) != 1 {
			return 0, ErrDenied
		}
		id = items[0].SourceID
	}
	if in.ShowID > 0 {
		var source int64
		if err := s.db.QueryRowContext(ctx, `SELECT source_id FROM tv_shows WHERE id=?`, in.ShowID).Scan(&source); err != nil {
			return 0, ErrDenied
		}
		id = source
	}
	if !p.Allows(id) {
		return 0, ErrDenied
	}
	var enabled bool
	if err := s.db.QueryRowContext(ctx, `SELECT enabled FROM sources WHERE id=?`, id).Scan(&enabled); err != nil || !enabled {
		return 0, ErrDenied
	}
	return id, nil
}
func (s *Service) QueueTask(ctx context.Context, p tokens.Token, tool string, in TaskInput) (Queued, error) {
	if !p.Has("jobs:write") || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 128 {
		return Queued{}, ErrDenied
	}
	switch tool {
	case "scan_source":
		if in.SourceID <= 0 || in.MediaID != 0 || in.ShowID != 0 {
			return Queued{}, ErrInvalid
		}
	case "search_media_candidates", "scrape_media_candidates", "scrape_artwork_candidates":
		if in.MediaID <= 0 || len(in.Query) > 256 {
			return Queued{}, ErrInvalid
		}
	case "scrape_tv_artwork_candidates", "scrape_tv_season", "scrape_tv_episode":
		if in.ShowID <= 0 || in.Season < 0 || in.Season > 1000 {
			return Queued{}, ErrInvalid
		}
		if tool == "scrape_tv_episode" && (in.Episode == nil || *in.Episode < 0) {
			return Queued{}, ErrInvalid
		}
	default:
		return Queued{}, ErrInvalid
	}
	if tool != "scan_source" && !p.Has("metadata:write") {
		return Queued{}, ErrDenied
	}
	source, err := s.source(ctx, p, in)
	if err != nil {
		return Queued{}, err
	}
	payload := taskPayload{p.ID, tool, in}
	return s.queue(ctx, p, tool, in.IdempotencyKey, hash(payload), source, "automation_task", payload, nil)
}
func (s *Service) queue(ctx context.Context, p tokens.Token, tool, key, digest string, source int64, kind string, payload any, prepare func(*sql.Tx) error) (Queued, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Queued{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE automation_requests SET created_at=created_at WHERE 0`); err != nil {
		return Queued{}, err
	}
	var existingHash string
	var existing Queued
	err = tx.QueryRowContext(ctx, `SELECT a.request_hash,j.id,j.state FROM automation_requests a JOIN jobs j ON j.id=a.job_id WHERE a.token_id=? AND a.tool=? AND a.idempotency_key=?`, p.ID, tool, key).Scan(&existingHash, &existing.JobID, &existing.State)
	if err == nil {
		if existingHash != digest {
			return Queued{}, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Queued{}, err
	}
	var backlog int
	if err = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM system_events WHERE dispatched_at IS NULL)+(SELECT COUNT(*) FROM webhook_deliveries WHERE state IN ('queued','retry_wait','delivering'))`).Scan(&backlog); err != nil {
		return Queued{}, err
	}
	if backlog >= s.backlogLimit {
		return Queued{}, errors.New("RATE_LIMITED")
	}
	var active int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE actor_token_id=? AND state IN ('queued','running')`, p.ID).Scan(&active); err != nil {
		return Queued{}, err
	}
	if active > 0 {
		return Queued{}, errors.New("RATE_LIMITED")
	}
	var valid bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM api_tokens WHERE id=? AND revoked_at IS NULL AND expires_at>?)`, p.ID, time.Now().UnixMilli()).Scan(&valid); err != nil || !valid {
		return Queued{}, ErrDenied
	}
	if prepare != nil {
		if err = prepare(tx); err != nil {
			return Queued{}, err
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Queued{}, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO jobs(kind,source_id,state,payload,max_retries,actor_token_id) VALUES(?,?,'queued',?,0,?)`, kind, source, string(body), p.ID)
	if err != nil {
		return Queued{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Queued{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO job_events(job_id,state,message) VALUES(?,'queued','Automation queued')`, id); err != nil {
		return Queued{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO automation_requests(token_id,tool,idempotency_key,request_hash,job_id,created_at) VALUES(?,?,?,?,?,?)`, p.ID, tool, key, digest, id, time.Now().UnixMilli()); err != nil {
		return Queued{}, err
	}
	if kind == "automation_apply" {
		if _, err = tx.ExecContext(ctx, `UPDATE automation_plans SET job_id=? WHERE id=?`, id, payload.(planPayload).PlanID); err != nil {
			return Queued{}, err
		}
	}
	return Queued{id, "queued"}, tx.Commit()
}
func (s *Service) runTask(ctx context.Context, job jobs.Job, progress func(int, string)) error {
	var payload taskPayload
	if json.Unmarshal([]byte(job.Payload), &payload) != nil {
		return ErrInvalid
	}
	p, err := s.tokens.Current(ctx, payload.TokenID)
	if err != nil || !p.Has("jobs:write") {
		return ErrDenied
	}
	in := payload.Input
	if _, err = s.source(ctx, p, in); err != nil {
		return err
	}
	if payload.Tool != "scan_source" && !p.Has("metadata:write") {
		return ErrDenied
	}
	ctx, cancel := s.watchRevocation(ctx, p.ID)
	defer cancel()
	var result any
	switch payload.Tool {
	case "scan_source":
		return s.library.ScanForAutomation(ctx, job.ID, in.SourceID, progress)
	case "search_media_candidates", "scrape_media_candidates":
		query := in.Query
		if query == "" {
			items, err := s.library.IntegrationMedia(ctx, p.SourceIDs, in.MediaID, 0, "", 1)
			if err != nil || len(items) != 1 {
				return ErrDenied
			}
			query = items[0].Title
		}
		candidates, err := s.metadata.Search(ctx, query, nil)
		if err != nil {
			return errors.New("PROVIDER_FAILED")
		}
		type candidate struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Year  *int   `json:"year"`
		}
		items := []candidate{}
		for _, c := range candidates {
			items = append(items, candidate{c.ID, c.Title, c.Year})
		}
		result = map[string]any{"candidates": items}
	case "scrape_artwork_candidates":
		items, err := s.metadata.ArtworkCandidates(ctx, in.MediaID)
		if err != nil {
			return errors.New("PROVIDER_FAILED")
		}
		result = map[string]int{"candidateCount": len(items)}
	case "scrape_tv_artwork_candidates":
		scope := "show"
		var season *int
		if in.Season > 0 {
			scope = "season"
			season = &in.Season
		}
		items, err := s.metadata.TVArtworkCandidates(ctx, in.ShowID, scope, season)
		if err != nil {
			return errors.New("PROVIDER_FAILED")
		}
		result = map[string]int{"candidateCount": len(items)}
	case "scrape_tv_season", "scrape_tv_episode":
		var episode *int
		if payload.Tool == "scrape_tv_episode" {
			episode = in.Episode
		}
		record, err := s.metadata.ScrapeTVMetadata(ctx, in.ShowID, in.Season, episode)
		if err != nil {
			return errors.New("PROVIDER_FAILED")
		}
		result = map[string]int{"episodeCount": len(record.Episodes)}
	default:
		return ErrInvalid
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if len(body) > 128<<10 {
		return errors.New("RESULT_TOO_LARGE")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO automation_results(job_id,result_json) VALUES(?,?) ON CONFLICT(job_id) DO UPDATE SET result_json=excluded.result_json`, job.ID, string(body))
	return err
}
func (s *Service) watchRevocation(ctx context.Context, id string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.tokens.Current(ctx, id); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel
}
func (s *Service) Cancel(ctx context.Context, p tokens.Token, id int64) (Queued, error) {
	if !p.Has("jobs:write") {
		return Queued{}, ErrDenied
	}
	var actor string
	if err := s.db.QueryRowContext(ctx, `SELECT actor_token_id FROM jobs WHERE id=?`, id).Scan(&actor); err != nil || actor != p.ID {
		return Queued{}, ErrDenied
	}
	job, err := s.library.CancelJob(ctx, id)
	return Queued{job.ID, job.State}, err
}
func (s *Service) RetryTask(ctx context.Context, p tokens.Token, id int64, key string) (Queued, error) {
	var body, kind, state, actor string
	if err := s.db.QueryRowContext(ctx, `SELECT payload,kind,state,actor_token_id FROM jobs WHERE id=?`, id).Scan(&body, &kind, &state, &actor); err != nil || actor != p.ID || kind != "automation_task" {
		return Queued{}, ErrDenied
	}
	if state != "failed" && state != "cancelled" && state != "interrupted" {
		return Queued{}, ErrConflict
	}
	var payload taskPayload
	if json.Unmarshal([]byte(body), &payload) != nil {
		return Queued{}, ErrInvalid
	}
	payload.Input.IdempotencyKey = key
	return s.QueueTask(ctx, p, payload.Tool, payload.Input)
}
func (s *Service) Result(ctx context.Context, p tokens.Token, id int64) (json.RawMessage, error) {
	var sourceID int64
	if err := s.db.QueryRowContext(ctx, `SELECT source_id FROM jobs WHERE id=? AND actor_token_id=?`, id, p.ID).Scan(&sourceID); err != nil || !p.Allows(sourceID) {
		return nil, ErrDenied
	}
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT r.result_json FROM automation_results r JOIN jobs j ON j.id=r.job_id WHERE j.id=? AND j.actor_token_id=?`, id, p.ID).Scan(&raw); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

func (s *Service) SetBacklogLimit(limit int) {
	if limit > 0 {
		s.backlogLimit = limit
	}
}
func (s *Service) Cleanup(ctx context.Context) error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := s.db.ExecContext(ctx, `DELETE FROM automation_results WHERE job_id IN (SELECT id FROM jobs WHERE state IN ('succeeded','failed','cancelled','interrupted') AND completed_at<? LIMIT 500)`, cutoff); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM automation_requests WHERE job_id IN (SELECT id FROM jobs WHERE state IN ('succeeded','failed','cancelled','interrupted') AND completed_at<? LIMIT 500)`, cutoff)
	return err
}

type ArtworkCandidate struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Language string `json:"language"`
}

func (s *Service) ArtworkCandidates(ctx context.Context, p tokens.Token, mediaID int64) ([]ArtworkCandidate, error) {
	if !p.Has("metadata:read") {
		return nil, ErrDenied
	}
	if _, err := s.source(ctx, p, TaskInput{MediaID: mediaID}); err != nil {
		return nil, err
	}
	items, err := s.metadata.CachedArtworkCandidates(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	result := []ArtworkCandidate{}
	for _, item := range items {
		if len(result) >= 100 {
			break
		}
		result = append(result, ArtworkCandidate{item.ID, item.Kind, item.Width, item.Height, item.Language})
	}
	return result, nil
}
