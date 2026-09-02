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
	StateQueued      = "queued"
	StateRunning     = "running"
	StateSucceeded   = "succeeded"
	StateFailed      = "failed"
	StateCancelled   = "cancelled"
	StateInterrupted = "interrupted"
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
	Payload         string `json:"-"`
	RetryCount      int    `json:"retryCount"`
	MaxRetries      int    `json:"maxRetries"`
	CreatedAt       string `json:"createdAt"`
	StartedAt       string `json:"startedAt,omitempty"`
	CompletedAt     string `json:"completedAt,omitempty"`
	UpdatedAt       string `json:"updatedAt"`
}

type Event struct {
	ID              int64  `json:"id"`
	JobID           int64  `json:"jobId"`
	State           string `json:"state"`
	ProgressCurrent int    `json:"progressCurrent"`
	ProgressTotal   int    `json:"progressTotal"`
	Message         string `json:"message"`
	CreatedAt       string `json:"createdAt"`
}

type HandlerFunc func(ctx context.Context, job Job, updateProgress func(current int, message string)) error

type Service struct {
	db       *sql.DB
	logger   *slog.Logger
	handlers map[string]HandlerFunc
	cancels  map[int64]context.CancelFunc
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
		cancels:  make(map[int64]context.CancelFunc),
	}
}

func (s *Service) RegisterHandler(kind string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[kind] = handler
}

func (s *Service) Queue(ctx context.Context, kind string, sourceID *int64) (Job, error) {
	return s.QueuePayload(ctx, kind, sourceID, []byte(`{}`))
}

func (s *Service) QueuePayload(ctx context.Context, kind string, sourceID *int64, payload []byte) (Job, error) {
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO jobs(kind, source_id, state, payload, max_retries) VALUES(?, ?, ?, ?, 2)`, kind, sourceID, StateQueued, string(payload))
	if err != nil {
		return Job{}, fmt.Errorf("queue job: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Job{}, fmt.Errorf("get job id: %w", err)
	}
	job, err := s.Get(ctx, id)
	if err == nil {
		s.recordEvent(ctx, job, "Job queued")
	}
	return job, err
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, payload, retry_count, max_retries, created_at, COALESCE(started_at,''), COALESCE(completed_at,''), updated_at FROM jobs ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	items := make([]Job, 0)
	for rows.Next() {
		var job Job
		var sourceID sql.NullInt64
		var message, errorMsg sql.NullString
		if err := rows.Scan(&job.ID, &job.Kind, &sourceID, &job.State, &job.ProgressCurrent, &job.ProgressTotal, &message, &errorMsg, &job.Payload, &job.RetryCount, &job.MaxRetries, &job.CreatedAt, &job.StartedAt, &job.CompletedAt, &job.UpdatedAt); err != nil {
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
	err := s.db.QueryRowContext(ctx, `SELECT id, kind, source_id, state, progress_current, progress_total, message, error_message, payload, retry_count, max_retries, created_at, COALESCE(started_at,''), COALESCE(completed_at,''), updated_at FROM jobs WHERE id=?`, id).
		Scan(&job.ID, &job.Kind, &sourceID, &job.State, &job.ProgressCurrent, &job.ProgressTotal, &message, &errorMsg, &job.Payload, &job.RetryCount, &job.MaxRetries, &job.CreatedAt, &job.StartedAt, &job.CompletedAt, &job.UpdatedAt)
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

func (s *Service) Cancel(ctx context.Context, id int64) (Job, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.State != StateQueued && job.State != StateRunning {
		return Job{}, errors.New("only queued or running jobs can be cancelled")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE jobs SET state='cancelled', message='Cancellation requested', error_message='', completed_at=datetime('now'), updated_at=datetime('now') WHERE id=? AND state IN ('queued','running')`, id)
	if err != nil {
		return Job{}, fmt.Errorf("cancel job: %w", err)
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return Job{}, errors.New("job state changed before cancellation")
	}
	s.mu.RLock()
	cancel := s.cancels[id]
	s.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
	job, err = s.Get(ctx, id)
	if err == nil {
		s.recordEvent(ctx, job, job.Message)
		s.logger.Info("job cancellation requested", "job_id", id, "kind", job.Kind)
	}
	return job, err
}

func (s *Service) Retry(ctx context.Context, id int64) (Job, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.State != StateFailed && job.State != StateCancelled && job.State != StateInterrupted {
		return Job{}, errors.New("only failed, cancelled, or interrupted jobs can be retried")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE jobs SET state='queued', progress_current=0, message='Retry queued', error_message='', started_at=NULL, completed_at=NULL, next_run_at=NULL, retry_count=retry_count+1, updated_at=datetime('now') WHERE id=?`, id)
	if err != nil {
		return Job{}, fmt.Errorf("retry job: %w", err)
	}
	job, err = s.Get(ctx, id)
	if err == nil {
		s.recordEvent(ctx, job, job.Message)
		s.logger.Info("job retry queued", "job_id", id, "kind", job.Kind, "retry_count", job.RetryCount)
	}
	return job, err
}

func (s *Service) EventsAfter(ctx context.Context, cursor int64, limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, job_id, state, progress_current, progress_total, message, created_at FROM job_events WHERE id>? ORDER BY id LIMIT ?`, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list job events: %w", err)
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.JobID, &event.State, &event.ProgressCurrent, &event.ProgressTotal, &event.Message, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Service) recordEvent(ctx context.Context, job Job, message string) {
	_, _ = s.db.ExecContext(ctx, `INSERT INTO job_events(job_id,state,progress_current,progress_total,message) VALUES(?,?,?,?,?)`, job.ID, job.State, job.ProgressCurrent, job.ProgressTotal, message)
}

func (s *Service) RunWorker(ctx context.Context) {
	// A process restart cannot safely resume a handler halfway through a file
	// rename. Mark abandoned work explicitly; queued jobs remain idempotent and
	// can be retried by the operator or a later recovery policy.
	_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='interrupted', updated_at=datetime('now'), completed_at=datetime('now'), error_message='worker stopped before completion' WHERE state='running'`)
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
	err := s.db.QueryRowContext(ctx, `SELECT id, kind FROM jobs WHERE state='queued' AND (next_run_at IS NULL OR next_run_at <= datetime('now')) ORDER BY id LIMIT 1`).Scan(&id, &kind)
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
		if failed, getErr := s.Get(ctx, id); getErr == nil {
			s.recordEvent(ctx, failed, failed.ErrorMessage)
		}
		return
	}

	s.logger.Info("job started", "job_id", id, "kind", kind)
	s.recordEvent(ctx, job, job.Message)
	jobCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancels[id] = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		delete(s.cancels, id)
		s.mu.Unlock()
	}()
	progressFn := func(current int, message string) {
		result, _ := s.db.ExecContext(ctx, `UPDATE jobs SET progress_current=?, message=?, updated_at=datetime('now') WHERE id=? AND state='running'`, current, message, id)
		if changed, _ := result.RowsAffected(); changed > 0 {
			if currentJob, getErr := s.Get(ctx, id); getErr == nil {
				s.recordEvent(ctx, currentJob, message)
			}
		}
	}

	if execErr := handler(jobCtx, job, progressFn); execErr != nil {
		if cancelled, getErr := s.Get(ctx, id); getErr == nil && cancelled.State == StateCancelled {
			return
		}
		if kind == "artwork_download" && job.RetryCount < job.MaxRetries && !errors.Is(execErr, context.Canceled) {
			_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='queued', retry_count=retry_count+1, error_message=?, next_run_at=datetime('now', '+5 seconds'), updated_at=datetime('now') WHERE id=?`, execErr.Error(), id)
			s.logger.Warn("job retry scheduled", "job_id", id, "kind", kind, "retry_count", job.RetryCount+1)
			if retried, getErr := s.Get(ctx, id); getErr == nil {
				s.recordEvent(ctx, retried, "Automatic retry scheduled")
			}
			return
		}
		state := StateFailed
		if errors.Is(execErr, context.Canceled) {
			state = StateCancelled
		}
		_, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state=?, error_message=?, completed_at=datetime('now'), updated_at=datetime('now') WHERE id=?`, state, execErr.Error(), id)
		s.logger.Error("job failed", "job_id", id, "kind", kind, "error", execErr)
		if failed, getErr := s.Get(ctx, id); getErr == nil {
			s.recordEvent(ctx, failed, failed.ErrorMessage)
		}
		return
	}

	result, _ = s.db.ExecContext(ctx, `UPDATE jobs SET state='succeeded', message='Job completed', completed_at=datetime('now'), updated_at=datetime('now') WHERE id=? AND state='running'`, id)
	if changed, _ := result.RowsAffected(); changed == 0 {
		return
	}
	s.logger.Info("job completed", "job_id", id, "kind", kind)
	if finished, getErr := s.Get(ctx, id); getErr == nil {
		s.recordEvent(ctx, finished, finished.Message)
	}
}
