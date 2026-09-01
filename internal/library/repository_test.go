package library

import (
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func newTestLibraryRepo(t *testing.T) (Repository, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "lib_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return NewRepository(db), dir
}

func TestLibraryRepositorySources(t *testing.T) {
	repo, dir := newTestLibraryRepo(t)
	ctx := t.Context()

	source, err := repo.CreateSource(ctx, "Movies", dir)
	if err != nil {
		t.Fatalf("CreateSource failed: %v", err)
	}
	if source.Name != "Movies" || source.RootPath != dir || !source.Enabled {
		t.Fatalf("unexpected created source: %+v", source)
	}

	updated, err := repo.UpdateSource(ctx, source.ID, "Movies HDD", dir, false)
	if err != nil || updated.Name != "Movies HDD" || updated.Enabled {
		t.Fatalf("UpdateSource failed: %v, got %+v", err, updated)
	}

	list, err := repo.ListSources(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListSources failed: %v, got %+v", err, list)
	}

	if err := repo.DeleteSource(ctx, source.ID); err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}

	if _, err := repo.GetSource(ctx, source.ID); err == nil {
		t.Fatal("expected error getting deleted source")
	}
}

func TestLibraryRepositoryMediaAndTV(t *testing.T) {
	repo, dir := newTestLibraryRepo(t)
	ctx := t.Context()

	source, err := repo.CreateSource(ctx, "Mixed", dir)
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	sqliteRepo := repo.(*sqliteRepository)
	_, err = sqliteRepo.db.ExecContext(ctx, `INSERT INTO media_items(source_id, relative_path, title_hint, year_hint, file_size, modified_at) VALUES(?, 'movie.mkv', 'Test Movie', 2024, 5000, datetime('now'))`, source.ID)
	if err != nil {
		t.Fatalf("insert movie: %v", err)
	}

	page, err := repo.ListMedia(ctx, "Test", 1, 10)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("ListMedia failed: %v, got %+v", err, page)
	}

	item, err := repo.GetMediaItem(ctx, page.Items[0].ID)
	if err != nil || item.TitleHint != "Test Movie" {
		t.Fatalf("GetMediaItem failed: %v, got %+v", err, item)
	}

	loc, err := repo.GetMediaLocation(ctx, item.ID)
	if err != nil || loc.AbsolutePath != filepath.Join(dir, "movie.mkv") {
		t.Fatalf("GetMediaLocation failed: %v, got %+v", err, loc)
	}
}
