package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestScansReconcileDeletedMoviesAndEpisodesOnlyOnSuccess(t *testing.T) {
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
			s := NewService(db, []string{root})
			source, err := s.CreateSource(t.Context(), "Fixture", root)
			if err != nil {
				t.Fatal(err)
			}
			files := []string{"Movie.mkv", "Show/Season 01/Show.S01E01.mkv", "Show/Season 01/Show.S01E02.mkv"}
			for _, name := range files {
				path := filepath.Join(root, name)
				if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
				t.Fatal(err)
			}
			// A disconnected/unavailable source must not hide the previous catalog.
			offline := root + "-offline"
			if err = os.Rename(root, offline); err != nil {
				t.Fatal(err)
			}
			scanErr := s.scan(t.Context(), 0, source.ID, mode, nil)
			if err = os.Rename(offline, root); err != nil {
				t.Fatal(err)
			}
			if scanErr == nil {
				t.Fatal("inaccessible source accepted")
			}
			var active int
			if err = db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=0`).Scan(&active); err != nil {
				t.Fatal(err)
			}
			if active != 3 {
				t.Fatalf("failed scan hid records: %d", active)
			}
			cancelled, cancel := context.WithCancel(t.Context())
			cancel()
			if err = s.scan(cancelled, 0, source.ID, mode, nil); err == nil {
				t.Fatal("cancelled scan succeeded")
			}
			for _, name := range files[:2] {
				if err = os.Remove(filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
				t.Fatal(err)
			}
			movies, err := s.ListMedia(t.Context(), "", 1, 50)
			if err != nil || movies.Total != 0 {
				t.Fatalf("deleted movie visible: %+v %v", movies, err)
			}
			shows, err := s.ListTVShows(t.Context(), "")
			if err != nil || len(shows) != 1 || shows[0].EpisodeCount != 1 {
				t.Fatalf("deleted episode counted: %+v %v", shows, err)
			}
			detail, err := s.TVShow(t.Context(), shows[0].ID)
			if err != nil || len(detail.Episodes) != 1 || detail.Episodes[0].EpisodeStart != 2 {
				t.Fatalf("stale episode detail %+v %v", detail, err)
			}
			if err = os.Remove(filepath.Join(root, files[2])); err != nil {
				t.Fatal(err)
			}
			if err = s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
				t.Fatal(err)
			}
			shows, err = s.ListTVShows(t.Context(), "")
			if err != nil || len(shows) != 0 {
				t.Fatalf("empty deleted show visible: %+v %v", shows, err)
			}
			var retained int
			if err = db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=1`).Scan(&retained); err != nil {
				t.Fatal(err)
			}
			if retained != 3 {
				t.Fatal("history removed rather than marking missing")
			}
		})
	}
}
