package library

import (
	"context"
	"errors"
	"fmt"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadScanBatchBoundsOrderAndCancellation(t *testing.T) {
	for _, readers := range []int{1, 2, 4} {
		t.Run(fmt.Sprint(readers), func(t *testing.T) {
			var active, peak atomic.Int32
			results := make([]int, scanBatchSize)
			first, last := errors.New("first"), errors.New("last")
			err := readScanBatch(t.Context(), len(results), readers, func(i int) error {
				n := active.Add(1)
				defer active.Add(-1)
				for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
				}
				time.Sleep(time.Duration(4-i%4) * time.Microsecond)
				results[i] = i + 1
				if i == 0 {
					return first
				}
				if i == 199 {
					return last
				}
				return nil
			})
			if !errors.Is(err, first) || peak.Load() > int32(readers) || active.Load() != 0 {
				t.Fatalf("err=%v peak=%d active=%d", err, peak.Load(), active.Load())
			}
			for i, result := range results {
				if result != i+1 {
					t.Fatal("unordered result", i, result)
				}
			}
			ctx, cancel := context.WithCancel(t.Context())
			started, release := make(chan struct{}, readers), make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- readScanBatch(ctx, 200, readers, func(int) error { started <- struct{}{}; <-release; return nil })
			}()
			for i := 0; i < readers; i++ {
				<-started
			}
			cancel()
			select {
			case <-done:
				t.Fatal("returned with live readers")
			default:
			}
			close(release)
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestScanReadersDeterministicCrossBatchHints(t *testing.T) {
	for _, readers := range []int{1, 2, 4} {
		t.Run(fmt.Sprint(readers), func(t *testing.T) {
			s, source, root, _ := scanBatchFixture(t, 0)
			s.scanReaders = readers
			for i := 0; i < 401; i++ {
				snapshotFixture(t, root, fmt.Sprintf("Show%03d.S01E001.Title.mkv", i))
			}
			for _, mode := range []string{"full", "incremental"} {
				if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
					t.Fatal(err)
				}
				shows, err := s.ListTVShows(t.Context(), "")
				if err != nil || len(shows) != 1 || shows[0].TitleHint != "Show400" || shows[0].EpisodeCount != 401 {
					t.Fatalf("%+v %v", shows, err)
				}
				var first, last string
				if err := s.db.QueryRow(`SELECT relative_path FROM media_items ORDER BY id LIMIT 1`).Scan(&first); err != nil {
					t.Fatal(err)
				}
				if err := s.db.QueryRow(`SELECT relative_path FROM media_items ORDER BY id DESC LIMIT 1`).Scan(&last); err != nil {
					t.Fatal(err)
				}
				if first != "Show000.S01E001.Title.mkv" || last != "Show400.S01E001.Title.mkv" {
					t.Fatal(first, last)
				}
			}
		})
	}
}

// Same tiny local index-only fixture as stage2, now varying only reader count.
func BenchmarkScanReaders(b *testing.B) {
	for _, readers := range []int{1, 2, 4} {
		for _, phase := range []string{"cold", "warm"} {
			b.Run(fmt.Sprintf("%d/%s", readers, phase), func(b *testing.B) {
				s, source, _, _ := scanBatchFixture(b, 10000)
				s.scanReaders = readers
				if phase == "warm" {
					if err := s.scan(b.Context(), 0, source.ID, "incremental", nil); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := s.scan(b.Context(), 0, source.ID, "incremental", nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// Valid local NFOs exercise bounded parsing plus ordered conditional persistence.
func BenchmarkScanReadersNFO(b *testing.B) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(previous) })
	for _, readers := range []int{1, 2, 4} {
		b.Run(fmt.Sprint(readers), func(b *testing.B) {
			s, source, root, _ := scanBatchFixture(b, 1000)
			s.scanReaders = readers
			s.SetMetadataHydrator(metadata.NewService(s.db, nil))
			entries, err := os.ReadDir(root)
			if err != nil {
				b.Fatal(err)
			}
			body := []byte("<movie><title>Fixture</title><plot>" + strings.Repeat("valid local plot ", 4000) + "</plot></movie>")
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".mkv" {
					if err := os.WriteFile(filepath.Join(root, strings.TrimSuffix(entry.Name(), ".mkv")+".nfo"), body, 0600); err != nil {
						b.Fatal(err)
					}
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := s.scan(b.Context(), 0, source.ID, "incremental", nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
