package jobs

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func newTestJobsService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "jobs_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return NewService(db, nil)
}

func TestQueueAndListJobs(t *testing.T) {
	service := newTestJobsService(t)
	ctx := t.Context()

	res, err := service.db.ExecContext(ctx, `INSERT INTO sources(name, root_path) VALUES('Test', '/media')`)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	sourceID, _ := res.LastInsertId()
	job, err := service.Queue(ctx, "scan", &sourceID)
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}
	if job.ID == 0 || job.Kind != "scan" || job.State != StateQueued || *job.SourceID != sourceID {
		t.Fatalf("unexpected job queued: %+v", job)
	}

	list, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != job.ID {
		t.Fatalf("unexpected job list: %+v", list)
	}
}

func TestWorkerExecutesHandlerSuccessfully(t *testing.T) {
	service := newTestJobsService(t)
	ctx := t.Context()

	var executed atomic.Bool
	service.RegisterHandler("test_job", func(ctx context.Context, job Job, updateProgress func(current int, message string)) error {
		updateProgress(50, "Halfway done")
		executed.Store(true)
		return nil
	})

	job, err := service.Queue(ctx, "test_job", nil)
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// Run single iteration
	service.runOne(ctx)

	if !executed.Load() {
		t.Fatal("expected handler to be executed")
	}

	finished, err := service.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get job failed: %v", err)
	}
	if finished.State != StateSucceeded || finished.ProgressCurrent != 50 {
		t.Fatalf("unexpected finished job state: %+v", finished)
	}
}

func TestWorkerHandlesHandlerFailure(t *testing.T) {
	service := newTestJobsService(t)
	ctx := t.Context()

	service.RegisterHandler("failing_job", func(ctx context.Context, job Job, updateProgress func(current int, message string)) error {
		return errors.New("simulated task failure")
	})

	job, err := service.Queue(ctx, "failing_job", nil)
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	service.runOne(ctx)

	failed, err := service.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get failed job failed: %v", err)
	}
	if failed.State != StateFailed || failed.ErrorMessage != "simulated task failure" {
		t.Fatalf("unexpected failed job: %+v", failed)
	}
}

func TestWorkerHandlesUnregisteredJobKind(t *testing.T) {
	service := newTestJobsService(t)
	ctx := t.Context()

	job, err := service.Queue(ctx, "unknown_kind", nil)
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	service.runOne(ctx)

	failed, err := service.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get job failed: %v", err)
	}
	if failed.State != StateFailed {
		t.Fatalf("expected state failed for unknown handler, got %s", failed.State)
	}
}

func TestWorkerCancellation(t *testing.T) {
	service := newTestJobsService(t)
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		service.RunWorker(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit upon context cancellation")
	}
}

func TestCancelQueuedJobAndRetry(t *testing.T) {
	service := newTestJobsService(t)
	job, err := service.Queue(t.Context(), "test_job", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.Cancel(t.Context(), job.ID)
	if err != nil || cancelled.State != StateCancelled {
		t.Fatalf("cancel queued job: %+v err=%v", cancelled, err)
	}
	retried, err := service.Retry(t.Context(), job.ID)
	if err != nil || retried.State != StateQueued || retried.RetryCount != 1 {
		t.Fatalf("retry cancelled job: %+v err=%v", retried, err)
	}
	events, err := service.EventsAfter(t.Context(), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].State != StateQueued || events[1].State != StateCancelled || events[2].State != StateQueued {
		t.Fatalf("unexpected job events: %+v", events)
	}
}

func TestCancelRunningJobPropagatesContext(t *testing.T) {
	service := newTestJobsService(t)
	started := make(chan struct{})
	service.RegisterHandler("blocking_job", func(ctx context.Context, _ Job, _ func(int, string)) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	job, err := service.Queue(t.Context(), "blocking_job", nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { service.runOne(t.Context()); close(done) }()
	<-started
	if _, err := service.Cancel(t.Context(), job.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("running handler did not receive cancellation")
	}
	current, err := service.Get(t.Context(), job.ID)
	if err != nil || current.State != StateCancelled {
		t.Fatalf("unexpected cancelled state: %+v err=%v", current, err)
	}
}
