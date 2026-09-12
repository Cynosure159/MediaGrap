package library

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

// BenchmarkScanDirectorySnapshots measures read-only index scans on local tiny
// fixtures, not NAS throughput. Setup is excluded; SQLite durability is unchanged.
func BenchmarkScanDirectorySnapshots(b *testing.B) {
	for _, layout := range []string{"flat", "nested"} {
		for _, count := range []int{100, 1000, 10000} {
			b.Run(fmt.Sprintf("%s/%d", layout, count), func(b *testing.B) {
				root := b.TempDir()
				for i := 0; i < count; i++ {
					directory := root
					if layout == "nested" {
						directory = filepath.Join(root, fmt.Sprintf("Group%05d", i/100))
					}
					if err := os.MkdirAll(directory, 0750); err != nil {
						b.Fatal(err)
					}
					stem := fmt.Sprintf("Movie%05d.2024", i)
					if i%5 == 0 {
						stem = fmt.Sprintf("Show%05d.S01E%03d", i/100, i%100+1)
					}
					for _, ext := range []string{".mkv", ".nfo"} {
						if err := os.WriteFile(filepath.Join(directory, stem+ext), []byte("fixture"), 0600); err != nil {
							b.Fatal(err)
						}
					}
				}
				db, err := database.Open(filepath.Join(b.TempDir(), "scan.db"))
				if err != nil {
					b.Fatal(err)
				}
				defer db.Close()
				if err := database.Migrate(b.Context(), db); err != nil {
					b.Fatal(err)
				}
				s := NewService(db, []string{root})
				source, err := s.CreateSource(b.Context(), "Benchmark", root)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := s.scan(b.Context(), 0, source.ID, "full", nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func snapshotFixture(t testing.TB, root, relative string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestScanDirectorySnapshotsEnumerateOnce(t *testing.T) {
	for _, layout := range []string{"flat", "nested"} {
		for _, count := range []int{100, 1000, 10000} {
			t.Run(fmt.Sprintf("%s/%d", layout, count), func(t *testing.T) {
				root := t.TempDir()
				for i := 0; i < count; i++ {
					prefix := ""
					if layout == "nested" {
						prefix = fmt.Sprintf("Group%05d/", i/100)
					}
					snapshotFixture(t, root, fmt.Sprintf("%sMovie%05d.mkv", prefix, i))
				}
				reads, videos := 0, 0
				err := walkScanDirectories(t.Context(), root, func(path string) ([]fs.DirEntry, error) {
					reads++
					return os.ReadDir(path)
				}, func(string, bool) (bool, error) { return false, nil }, func(_ string, snapshot directorySnapshot) error {
					videos += snapshot.videos
					return nil
				})
				wantReads := 1
				if layout == "nested" {
					wantReads += count / 100
				}
				if err != nil || reads != wantReads || videos != count {
					t.Fatalf("reads=%d want=%d videos=%d want=%d: %v", reads, wantReads, videos, count, err)
				}
				t.Logf("videos=%d directory_enumerations=%d", videos, reads)
			})
		}
	}
}

func TestScanDirectorySnapshotsSidecarAttribution(t *testing.T) {
	root := t.TempDir()
	files := []string{"A[1].2024.mkv", "A[1].2024.nfo", "A[1].2024.poster.JPG", "A[1].2024.extended.mkv", "A[1].2024.extended.nfo", "poster.jpg", "Solo/Movie.mkv", "Solo/Movie-sample.mkv", "Solo/movie.nfo", "Solo/poster.jpg", "Solo/Extras/Bonus.mkv", "Show (2024)/Season 01/Show.S01E01.mkv", "Show (2024)/Season 01/Show.S01E01.nfo", "Show (2024)/Season 01/Show.S01E02.mkv", "Show (2024)/Season 01/Show.S01E02.nfo", "Show (2024)/Season 01/poster.jpg"}
	for _, name := range files {
		snapshotFixture(t, root, name)
	}
	if err := os.Symlink(filepath.Join(root, "Solo"), filepath.Join(root, "Linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "poster.jpg"), filepath.Join(root, "A[1].2024.link.jpg")); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "scan.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, []string{root})
	source, err := s.CreateSource(t.Context(), "Fixture", root)
	if err != nil {
		t.Fatal(err)
	}
	reads := map[string]int{}
	s.readDirectory = func(path string) ([]fs.DirEntry, error) { reads[path]++; return os.ReadDir(path) }
	if err := s.scan(t.Context(), 0, source.ID, "full", nil); err != nil {
		t.Fatal(err)
	}
	// Count the complete indexing path, including live sidecar discovery if it
	// is accidentally reintroduced. Read APIs below have a separate budget.
	wantReads := map[string]int{
		root:                               1,
		filepath.Join(root, "Solo"):        1,
		filepath.Join(root, "Show (2024)"): 1,
		filepath.Join(root, "Show (2024)", "Season 01"): 1,
	}
	if !reflect.DeepEqual(reads, wantReads) {
		t.Fatalf("scan directory enumeration budget: got %v want %v", reads, wantReads)
	}
	t.Log("complete service scan: 4 enumerations for 4 visited directories and 5 indexed videos")
	clear(reads)
	expected := map[string][]Sidecar{
		"A[1].2024.mkv":                         {{RelativePath: "A[1].2024.extended.nfo", Kind: "nfo"}, {RelativePath: "A[1].2024.nfo", Kind: "nfo"}, {RelativePath: "A[1].2024.poster.JPG", Kind: "image"}},
		"A[1].2024.extended.mkv":                {{RelativePath: "A[1].2024.extended.nfo", Kind: "nfo"}},
		"Solo/Movie.mkv":                        {{RelativePath: "Solo/movie.nfo", Kind: "nfo"}, {RelativePath: "Solo/poster.jpg", Kind: "image"}},
		"Show (2024)/Season 01/Show.S01E01.mkv": {{RelativePath: "Show (2024)/Season 01/Show.S01E01.nfo", Kind: "nfo"}},
		"Show (2024)/Season 01/Show.S01E02.mkv": {{RelativePath: "Show (2024)/Season 01/Show.S01E02.nfo", Kind: "nfo"}},
	}
	for relative, want := range expected {
		var id int64
		if err := db.QueryRow(`SELECT id FROM media_items WHERE relative_path=? AND missing=0`, relative).Scan(&id); err != nil {
			t.Fatal(err)
		}
		got, err := s.sidecars(t.Context(), id)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %+v want %+v err=%v", relative, got, want, err)
		}
		if live := s.currentSidecarsForMedia(root, relative); !reflect.DeepEqual(live, want) {
			t.Fatalf("live %s: %+v", relative, live)
		}
	}
	wantLiveReads := map[string]int{
		root:                        2,
		filepath.Join(root, "Solo"): 1,
		filepath.Join(root, "Show (2024)", "Season 01"): 2,
	}
	if !reflect.DeepEqual(reads, wantLiveReads) {
		t.Fatalf("live sidecar discovery bypassed shared enumeration: got %v want %v", reads, wantLiveReads)
	}
	movies, err := s.ListMedia(t.Context(), "", 1, 100)
	if err != nil || movies.Total != 3 {
		t.Fatalf("movies=%+v err=%v", movies, err)
	}
	shows, err := s.ListTVShows(t.Context(), "")
	if err != nil || len(shows) != 1 || shows[0].EpisodeCount != 2 || shows[0].TitleHint != "Show" || shows[0].YearHint == nil || *shows[0].YearHint != 2024 {
		t.Fatalf("shows=%+v err=%v", shows, err)
	}
	// Snapshots never outlive a scan; a removed sidecar disappears on rescan.
	if err := os.Remove(filepath.Join(root, "Solo/poster.jpg")); err != nil {
		t.Fatal(err)
	}
	clear(reads)
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reads, wantReads) {
		t.Fatalf("rescan directory enumeration budget: got %v want %v", reads, wantReads)
	}
	var stale int
	if err := db.QueryRow(`SELECT count(*) FROM sidecar_assets WHERE relative_path='Solo/poster.jpg'`).Scan(&stale); err != nil || stale != 0 {
		t.Fatalf("stale=%d err=%v", stale, err)
	}
}

func TestDirectorySnapshotRevalidatesSidecarType(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Movie.mkv", "movie.nfo", "poster.jpg"} {
		snapshotFixture(t, root, name)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := newDirectorySnapshot(entries)
	for _, name := range []string{"movie.nfo", "poster.jpg"} {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "Movie.mkv"), filepath.Join(root, "movie.nfo")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "poster.jpg"), 0750); err != nil {
		t.Fatal(err)
	}
	if got := snapshot.sidecars(root, "Movie.mkv"); len(got) != 0 {
		t.Fatalf("nonregular sidecars accepted: %+v", got)
	}
}

func TestScanSkippedHistoryReconcilesOnlyAfterSuccess(t *testing.T) {
	for _, mode := range []string{"full", "incremental"} {
		for _, failure := range []string{"cancel", "read-error"} {
			t.Run(mode+"/"+failure, func(t *testing.T) {
				root := t.TempDir()
				// The main movie is already gone; historical identity must still suppress Extras.
				files := []string{"Movie/Movie-sample.mkv", "Movie/Extras/Bonus.mkv", "ZFailure/Other.mkv"}
				for _, relative := range files {
					snapshotFixture(t, root, relative)
				}
				db, err := database.Open(filepath.Join(t.TempDir(), "scan.db"))
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if err := database.Migrate(t.Context(), db); err != nil {
					t.Fatal(err)
				}
				s := NewService(db, []string{root})
				source, err := s.CreateSource(t.Context(), "Fixture", root)
				if err != nil {
					t.Fatal(err)
				}
				for _, relative := range append(files[:2:2], "Movie/Main.mkv") {
					if _, err := db.Exec(`INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at,last_seen_at) VALUES(?,?,?,1,'old','old')`, source.ID, relative, relative); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := db.Exec(`UPDATE sources SET last_scan_at='2000-01-01 00:00:00'`); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				injected := errors.New("injected directory read error")
				s.readDirectory = func(path string) ([]fs.DirEntry, error) {
					if filepath.Base(path) == "ZFailure" {
						if failure == "cancel" {
							cancel()
						} else {
							return nil, injected
						}
					}
					return os.ReadDir(path)
				}
				err = s.scan(ctx, 0, source.ID, mode, nil)
				wantErr := injected
				if failure == "cancel" {
					wantErr = context.Canceled
				}
				if !errors.Is(err, wantErr) {
					t.Fatalf("scan err=%v want=%v", err, wantErr)
				}
				var hidden int
				if err := db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=1`).Scan(&hidden); err != nil || hidden != 0 {
					t.Fatalf("failed scan hid %d records: %v", hidden, err)
				}
				var lastScan string
				if err := db.QueryRow(`SELECT last_scan_at FROM sources`).Scan(&lastScan); err != nil || lastScan != "2000-01-01 00:00:00" {
					t.Fatalf("last scan advanced: %q %v", lastScan, err)
				}
				s.readDirectory = os.ReadDir
				if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=1`).Scan(&hidden); err != nil || hidden != 3 {
					t.Fatalf("successful scan retained %d historical rows: %v", hidden, err)
				}
				for _, relative := range files {
					data, err := os.ReadFile(filepath.Join(root, relative))
					if err != nil || string(data) != "fixture" {
						t.Fatalf("fixture mutated %s: %q %v", relative, data, err)
					}
				}
			})
		}
	}
}
