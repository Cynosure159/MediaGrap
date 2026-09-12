package library

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mediagrap/mediagrap/internal/metadata"
)

type orderedScanHydrator struct {
	active  atomic.Int32
	mu      sync.Mutex
	applied []int64
	invalid atomic.Bool
}

func (h *orderedScanHydrator) PrepareExistingNFO(ctx context.Context, id int64, _ string) (metadata.Record, bool, error) {
	h.active.Add(1)
	defer h.active.Add(-1)
	if id <= 0 {
		h.invalid.Store(true)
	}
	return metadata.Record{MediaItemID: id, Title: "Fixture"}, true, ctx.Err()
}
func (h *orderedScanHydrator) HydratePreparedNFO(ctx context.Context, record metadata.Record) error {
	if h.active.Load() != 0 {
		h.invalid.Store(true)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.applied = append(h.applied, record.MediaItemID)
	return ctx.Err()
}
func TestScanHydrationReadersSingleOrderedWriter(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 401)
	h := &orderedScanHydrator{}
	s.SetMetadataHydrator(h)
	s.scanReaders = 4
	// A Kodi fallback in this multi-video directory makes all movies eligible.
	snapshotFixture(t, root, "movie.nfo")
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	if h.invalid.Load() || len(h.applied) != 401 {
		t.Fatal(h.invalid.Load(), len(h.applied))
	}
	for i, id := range h.applied {
		if id != int64(i+1) {
			t.Fatalf("unordered ID %d at %d", id, i)
		}
	}
}
