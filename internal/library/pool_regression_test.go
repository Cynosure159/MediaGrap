package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestCatalogSingleConnection(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Movie.mkv", "Movie.nfo", "Show.S01E01.mkv"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
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
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Run("movies", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()
		page, err := s.ListMedia(ctx, "", 1, 30)
		if err != nil {
			t.Fatal(err)
		}
		if ctx.Err() != nil {
			t.Fatalf("nested query exhausted pool: %v", ctx.Err())
		}
		if len(page.Items) != 1 || len(page.Items[0].Sidecars) != 1 {
			t.Fatalf("missing catalog or sidecars: %+v", page)
		}
	})
	t.Run("tv", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()
		shows, err := s.ListTVShows(ctx, "")
		if err != nil || len(shows) != 1 {
			t.Fatalf("shows=%+v err=%v", shows, err)
		}
		detail, err := s.TVShow(ctx, shows[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		if ctx.Err() != nil {
			t.Fatalf("nested query exhausted pool: %v", ctx.Err())
		}
		if len(detail.Episodes) != 1 {
			t.Fatalf("episodes=%+v", detail)
		}
	})
	if db.Stats().InUse != 0 {
		t.Fatal("connections leaked")
	}
}
