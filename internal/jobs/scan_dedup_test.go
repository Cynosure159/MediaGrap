package jobs

import (
	"errors"
	"sync"
	"testing"
)

func TestConcurrentScanQueueAndRetry(t *testing.T) {
	s := newTestJobsService(t)
	result, err := s.db.Exec(`INSERT INTO sources(name,root_path) VALUES('Fixture','/fixture')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for range 12 {
		wg.Go(func() { _, err := s.Queue(t.Context(), "scan", &id); results <- err })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrScanAlreadyActive) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("queued %d scans", successes)
	}
	jobs, err := s.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Cancel(t.Context(), jobs[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Queue(t.Context(), "scan", &id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Retry(t.Context(), jobs[0].ID); err == nil {
		t.Fatal("retry bypassed scan dedup")
	}
}
