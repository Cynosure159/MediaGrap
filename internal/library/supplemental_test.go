package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestSampleVideoNames(t *testing.T) {
	for _, name := range []string{"Movie-sample.mkv", "Movie.SAMPLE.mp4", "Movie_sample.avi", "sample.mkv"} {
		if !isSampleVideo(name) {
			t.Errorf("not recognized: %s", name)
		}
	}
	for _, name := range []string{"The Sample (2024).mkv", "Sample.Movie.2020.mkv", "Movie-samples.mkv", "Samples.mkv", "Example.mkv"} {
		if isSampleVideo(name) {
			t.Errorf("false positive: %s", name)
		}
	}
}

func TestRescanExcludesSupplementalVideosWithoutDeletingFiles(t *testing.T) {
	for _, mode := range []string{"incremental", "full"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err = database.Migrate(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			service := NewService(db, []string{root})
			source, err := service.CreateSource(t.Context(), "Movies", root)
			if err != nil {
				t.Fatal(err)
			}
			main := "Movie [x%_]/Movie.mkv"
			extras := []string{"Movie [x%_]/Movie-sample.mkv", "Movie [x%_]/Extras/Moving Pictures.mkv", "Movie [x%_]/Extra/Trailer.mkv"}
			keep := []string{main, "Movie [x%_]/ExtrasMore/Other.mkv", "The Sample (2024)/The Sample (2024).mkv", "Extras/Extras.S01E01.mkv"}
			for _, relative := range append(append([]string{}, extras...), keep...) {
				path := filepath.Join(root, relative)
				if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
					t.Fatal(err)
				}
				// Simulate the old scanner having indexed all supplemental videos.
				if _, err = db.Exec(`INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,'Fixture',9,'now')`, source.ID, relative); err != nil {
					t.Fatal(err)
				}
			}
			nfoPath := filepath.Join(root, "Movie [x%_]/movie.nfo")
			if err = os.WriteFile(nfoPath, []byte("<movie><title>Movie</title></movie>"), 0600); err != nil {
				t.Fatal(err)
			}
			if err = service.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
				t.Fatal(err)
			}
			for _, relative := range append(append([]string{}, extras...), keep...) {
				var missing int
				if err = db.QueryRow(`SELECT missing FROM media_items WHERE source_id=? AND relative_path=?`, source.ID, relative).Scan(&missing); err != nil {
					t.Fatal(err)
				}
				excluded := false
				for _, extra := range extras {
					if relative == extra {
						excluded = true
					}
				}
				if (missing == 1) != excluded {
					t.Errorf("%s missing=%d excluded=%v", relative, missing, excluded)
				}
				b, err := os.ReadFile(filepath.Join(root, relative))
				if err != nil || string(b) != "unchanged" {
					t.Fatalf("file changed %s", relative)
				}
			}
			if !directoryHasOneVideo(filepath.Join(root, "Movie [x%_]")) {
				t.Fatal("sample still counted as a main video")
			}
			sidecars := service.currentSidecarsForMedia(root, main)
			if len(sidecars) != 1 || sidecars[0].Kind != "nfo" {
				t.Fatalf("directory NFO not attached to main: %+v", sidecars)
			}
			page, err := service.ListMedia(t.Context(), "", 1, 50)
			if err != nil || page.Total != 3 {
				t.Fatalf("wrong movie total %+v %v", page, err)
			}
			if err := os.Remove(filepath.Join(root, main)); err != nil {
				t.Fatal(err)
			}
			if err := service.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
				t.Fatal(err)
			}
			page, err = service.ListMedia(t.Context(), "", 1, 50)
			if err != nil || page.Total != 2 {
				t.Fatalf("extras revived after main deletion: %+v %v", page, err)
			}
			for _, extra := range extras {
				if b, err := os.ReadFile(filepath.Join(root, extra)); err != nil || string(b) != "unchanged" {
					t.Fatal("extra file changed")
				}
			}
		})
	}
}
