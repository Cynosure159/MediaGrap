package library

import (
	"fmt"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"path/filepath"
	"testing"
)

func TestCatalogBatchesHaveStableOrderAndGlobalFilters(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	s := NewService(db, []string{root})
	source, err := s.CreateSource(t.Context(), "Fixture", root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 137; i++ {
		quality := "1080p"
		if i%2 == 0 {
			quality = "4k"
		}
		_, err = db.Exec(`INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at,year_hint) VALUES(?,?,?,?,?,?)`, source.ID, fmt.Sprintf("Movie%03d.%s.mkv", i, quality), "Same title", i, "now", 1900+i)
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[int64]bool{}
	for page := 1; page <= 3; page++ {
		result, err := s.ListMedia(t.Context(), "", page, 50)
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 137 {
			t.Fatal(result.Total)
		}
		for _, item := range result.Items {
			if seen[item.ID] {
				t.Fatal("duplicate", item.ID)
			}
			seen[item.ID] = true
		}
	}
	if len(seen) != 137 {
		t.Fatal(len(seen))
	}
	result, err := s.ListMedia(t.Context(), "", 1, 50, CatalogOptions{Filter: "4k", Sort: "size"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 68 || result.Items[0].FileSize != 136 {
		t.Fatalf("bad global filter/order: total=%d first=%+v", result.Total, result.Items[0])
	}
	result, err = s.ListMedia(t.Context(), "", 1, 50, CatalogOptions{Sort: "year"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].FileSize != 137 {
		t.Fatal("not globally sorted")
	}
}
