package library

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Index-only, tiny local fixtures; neither a cold OS cache nor production NAS.
// Run with -benchtime=1x: each phase has a fresh migrated DB; warm/change phases
// seed it outside the timer. The same test/driver observer runs against stage 1.
func BenchmarkScanIncremental(b *testing.B) {
	for _, layout := range []string{"flat", "nested"} {
		for _, count := range []int{1000, 10000} {
			for _, phase := range []string{"cold", "warm", "one-percent-change"} {
				b.Run(fmt.Sprintf("%s/%d/%s", layout, count, phase), func(b *testing.B) {
					s, source, root, counts := scanBatchFixture(b, 0)
					names := make([]string, count)
					for i := 0; i < count; i++ {
						directory := ""
						if layout == "nested" {
							directory = fmt.Sprintf("Group%05d", i/100)
						}
						stem := fmt.Sprintf("Movie%05d.2024", i)
						if i%5 == 0 {
							stem = fmt.Sprintf("Show%05d.S01E%03d", i/100, i%100+1)
						}
						names[i] = filepath.Join(directory, stem+".mkv")
						snapshotFixture(b, root, names[i])
						snapshotFixture(b, root, filepath.Join(directory, stem+".nfo"))
					}
					if phase != "cold" {
						if err := s.scan(b.Context(), 0, source.ID, "incremental", nil); err != nil {
							b.Fatal(err)
						}
					}
					if phase == "one-percent-change" {
						for i := 0; i < count/100; i++ {
							if err := os.WriteFile(filepath.Join(root, names[i]), []byte("changed fixture"), 0600); err != nil {
								b.Fatal(err)
							}
						}
					}
					counts.reset()
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if err := s.scan(b.Context(), 0, source.ID, "incremental", nil); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					statements, transactions, _ := counts.snapshot()
					b.ReportMetric(float64(statements)/float64(b.N), "SQL/op")
					b.ReportMetric(float64(transactions)/float64(b.N), "tx/op")
				})
			}
		}
	}
}
