package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestPreviewAndApplyNeverOverwritesNFO(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	video := filepath.Join(root, "Example.2024.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, filepath.Base(video), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	year := 2024
	service := NewService(db, nil)
	plan, err := service.Preview(ctx, Record{MediaItemID: itemID, Title: "Example", Year: &year}, video, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Conflict {
		t.Fatal("new target should not conflict")
	}
	applied, err := service.Apply(ctx, plan.ID, func(path string) bool { return strings.HasPrefix(path, root+string(filepath.Separator)) })
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != "applied" {
		t.Fatalf("expected applied plan, got %s", applied.State)
	}
	content, err := os.ReadFile(strings.TrimSuffix(video, ".mkv") + ".nfo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<title>Example</title>") {
		t.Fatal("expected written NFO")
	}
	second, err := service.Preview(ctx, Record{MediaItemID: itemID, Title: "Changed"}, video, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, second.ID, func(string) bool { return true }); err == nil {
		t.Fatal("existing NFO must cause a conflict")
	}
	unchanged, err := os.ReadFile(strings.TrimSuffix(video, ".mkv") + ".nfo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(unchanged), "Changed") {
		t.Fatal("existing NFO was overwritten")
	}
}
