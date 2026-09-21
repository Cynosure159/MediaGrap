package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func supplementalFixture(t *testing.T, relative string) (*Service, int64, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(t.TempDir(), "supplemental.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(t.Context(), `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("video fixture"), 0640); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	result, err = db.ExecContext(t.Context(), `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, relative, "Movie", 2024, info.Size(), info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"))
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	service := NewService(db, []string{root})
	service.prober = &fakeMediaProber{}
	return service, itemID, path
}

func TestInspectMediaShowsDirectSamplesAndExtrasWithoutChangingMainAudit(t *testing.T) {
	service, itemID, mainPath := supplementalFixture(t, "Movie/Movie.mkv")
	parent := filepath.Dir(mainPath)
	for _, relative := range []string{"Movie-sample.MKV", "sample.mp4"} {
		if err := os.WriteFile(filepath.Join(parent, relative), []byte("sample"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	extraDir := filepath.Join(parent, "eXtRaS")
	if err := os.Mkdir(extraDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "Behind.m4v"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(extraDir, "Nested"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "Nested", "ignored.mkv"), []byte("nested"), 0600); err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.SupplementalStatus != "ready" || len(inspection.SupplementalFiles) != 3 {
		t.Fatalf("unexpected supplemental audit: %#v", inspection)
	}
	for _, file := range inspection.SupplementalFiles {
		if strings.Contains(file.RelativePath, "ignored") || file.MIMEType == "application/octet-stream" {
			t.Fatalf("nested or content-sniffed file included: %#v", file)
		}
	}
	for _, file := range inspection.Files {
		if strings.Contains(file.RelativePath, "sample") || strings.Contains(file.RelativePath, "Behind") {
			t.Fatalf("supplemental video leaked into main file audit: %#v", inspection.Files)
		}
	}
	preview, err := service.PreviewNaming(t.Context(), itemID, "${title}", NamingValues{Title: "Renamed"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range preview.Items {
		if strings.Contains(item.CurrentPath, "sample") || strings.Contains(item.CurrentPath, "Behind") {
			t.Fatalf("supplemental video leaked into rename plan: %#v", preview.Items)
		}
	}
}

func TestSupplementalAuditRejectsAmbiguityAndSymlink(t *testing.T) {
	service, itemID, mainPath := supplementalFixture(t, "Shared/Movie.mkv")
	if err := os.WriteFile(filepath.Join(filepath.Dir(mainPath), "Other.mkv"), []byte("other"), 0600); err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.SupplementalStatus != "not_associated" || len(inspection.SupplementalFiles) != 0 {
		t.Fatalf("expected ambiguous directory to remain empty: %#v", inspection)
	}
	if err := os.Remove(filepath.Join(filepath.Dir(mainPath), "Other.mkv")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(filepath.Dir(mainPath), "Extras"), 0750); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(filepath.Dir(mainPath), "Extras", "escape.mkv")); err != nil {
		t.Fatal(err)
	}
	inspection, err = service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.SupplementalStatus != "unreadable" || len(inspection.SupplementalFiles) != 0 {
		t.Fatalf("expected symlink to be rejected: %#v", inspection)
	}
}

func TestSupplementalAuditIsBoundedAndRefreshesCachedProbe(t *testing.T) {
	service, itemID, mainPath := supplementalFixture(t, "Movie/Movie.mkv")
	first, err := service.InspectMedia(t.Context(), itemID)
	if err != nil || first.SupplementalStatus != "ready" {
		t.Fatalf("unexpected first inspection: %#v %v", first, err)
	}
	if err := os.Mkdir(filepath.Join(filepath.Dir(mainPath), "Extras"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(mainPath), "Extras", "new.mkv"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := service.InspectMedia(t.Context(), itemID)
	if err != nil || !second.Cached || len(second.SupplementalFiles) != 1 {
		t.Fatalf("cached inspection did not refresh supplemental view: %#v %v", second, err)
	}

	for i := 0; i < supplementalDirectoryEntryBudget+1; i++ {
		name := filepath.Join(filepath.Dir(mainPath), "Extras", fmt.Sprintf("limit-%03d.mkv", i))
		if err := os.WriteFile(name, []byte("extra"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	limited, err := service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	if limited.SupplementalStatus != "truncated" || limited.SupplementalWarning == "" {
		t.Fatalf("expected bounded incomplete result: %#v", limited)
	}
}
