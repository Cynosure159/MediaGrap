package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// State constants for durable jobs
const (
	StateQueued    = "queued"
	StateRunning   = "running"
	StateSucceeded = "succeeded"
	StateFailed    = "failed"
	StateCancelled = "cancelled"
)

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

type HandlerFunc func(ctx context.Context, job Job, updateProgress func(current int, message string)) error

type Service struct {
	db       *sql.DB
	logger   *slog.Logger
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
}

func NewService(db *sql.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db:       db,
		logger:   logger,
		handlers: make(map[string]HandlerFunc),
	}
}

func (s *Service) RegisterHandler(kind string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[kind] = handler
}

func (s *Service) Queue(ctx context.Context, kind string, sourceID *int64) (Job, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO jobs(kind, source_id, state) VALUES(?, ?, ?)`, kind, sourceID, StateQueued)
	if err != nil {
		return Job{}, fmt.Errorf("queue job: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Job{}, fmt.Errorf("get job id: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, created_at FROM jobs ORDER BY id DESC LIMIT 50`)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	items := make([]Job, 0)
	for rows.Next() {
		var job Job
		var sourceID sql.NullInt64
		var message, errorMsg sql.NullString
		if err := rows.Scan(&job.ID, &job.Kind, &sourceID, &job.State, &job.ProgressCurrent, &job.ProgressTotal, &message, &errorMsg, &job.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		if sourceID.Valid {
			job.SourceID = &sourceID.Int64
		}
		if message.Valid {
			job.Message = message.String
		}
		if errorMsg.Valid {
			job.ErrorMessage = errorMsg.String
		}
		items = append(items, job)
	}
	return items, rows.Err()
}

func (s *Service) Get(ctx context.Context, id int64) (Job, error) {
	var job Job
	var sourceID sql.NullInt64
	var message, errorMsg sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, created_at FROM jobs WHERE id=?`, id).
		Scan(&job.ID, &job.Kind, &sourceID, &job.State, &job.ProgressCurrent, &job.ProgressTotal, &message, &errorMsg, &job.CreatedAt)
	if err != nil {
		return Job{}, errors.New("job not found")
	}
	if sourceID.Valid {
		job.SourceID = &sourceID.Int64
	}
	if message.Valid {
		job.Message = message.String
	}
	if errorMsg.Valid {
		job.ErrorMessage = errorMsg.String
	}
	return job, nil
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
	var id int64
	var kind string
	err := s.db.QueryRowContext(ctx, `SELECT id, kind FROM jobs WHERE state='queued' ORDER BY id LIMIT 1`).Scan(&id, &kind)
	if err != nil {
		return
	}

	result, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='running', started_at=datetime('now'), updated_at=datetime('now'), message='Job started' WHERE id=? AND state='queued'`, id)
	if err != nil {
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return
	}

	s.mu.RLock()
	handler, exists := s.handlers[kind]
	s.mu.RUnlock()

	job, err := s.Get(ctx, id)
	if err != nil {
		return
	}

	if !exists {
		_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='failed', error_message='no handler registered for job kind', completed_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, id)
		s.logger.Error("no handler for job", "job_id", id, "kind", kind)
		return
	}

	s.logger.Info("job started", "job_id", id, "kind", kind)
	progressFn := func(current int, message string) {
		_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET progress_current=?, message=?, updated_at=datetime('now') WHERE id=?`, current, message, id)
	}

	if execErr := handler(ctx, job, progressFn); execErr != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='failed', error_message=?, completed_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, execErr.Error(), id)
		s.logger.Error("job failed", "job_id", id, "kind", kind, "error", execErr)
		return
	}

	_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='succeeded', message='Job completed', completed_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, id)
	s.logger.Info("job completed", "job_id", id, "kind", kind)
}
