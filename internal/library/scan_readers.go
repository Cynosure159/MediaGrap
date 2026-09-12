package library

import (
	"context"
	"fmt"
	"sync"

	"github.com/mediagrap/mediagrap/internal/metadata"
)

// readerCount is instance-scoped for measurements; no process-global tuning.
func (s *Service) readerCount() int {
	if s.scanReaders >= 1 && s.scanReaders <= 4 {
		return s.scanReaders
	}
	return 2
}

// readScanBatch joins every reader before returning, including cancellation.
// Each reader owns distinct result slots; the caller consumes them in traversal
// order. A slow read backpressures the next chunk rather than growing a queue.
func readScanBatch(ctx context.Context, count, readers int, read func(int) error) error {
	if count > scanBatchSize || readers < 1 || readers > 4 {
		return fmt.Errorf("invalid scan reader batch")
	}
	errors := make([]error, count)
	var wg sync.WaitGroup
	for worker := 0; worker < min(readers, count); worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := worker; i < count; i += readers {
				if ctx.Err() != nil {
					return
				}
				errors[i] = read(i)
			}
		}(worker)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) hydrateScanBatch(ctx context.Context, batch []scanCandidate) error {
	if s.metadataHydrator == nil {
		return ctx.Err()
	}
	type prepared struct {
		record metadata.Record
		found  bool
		err    error
	}
	// Parsed XML can be much larger than a stat candidate. Retain only one
	// record per reader, not 200 parsed NFOs, before applying them in order.
	for start := 0; start < len(batch); start += s.readerCount() {
		window := batch[start:min(start+s.readerCount(), len(batch))]
		records := make([]prepared, len(window))
		if err := readScanBatch(ctx, len(window), s.readerCount(), func(i int) error {
			if window[i].hydrate {
				records[i].record, records[i].found, records[i].err = s.metadataHydrator.PrepareExistingNFO(ctx, window[i].id, window[i].path)
			}
			// Invalid local NFOs are non-fatal, as in the serial scanner.
			return nil
		}); err != nil {
			return err
		}
		for i, prepared := range records {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := prepared.err
			if err == nil && prepared.found {
				err = s.metadataHydrator.HydratePreparedNFO(ctx, prepared.record)
			}
			if err != nil {
				s.logger.Warn("existing movie NFO hydration failed", "media_item_id", window[i].id, "error", err)
			}
		}
	}
	return nil
}
