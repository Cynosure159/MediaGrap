package settings

import (
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestUpdateHidesAPIKeyAndPersistsLanguage(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/mediagrap.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, Defaults{TMDbLanguage: "en-US", MediaRoots: []string{"/media"}})
	if _, err := service.Update(t.Context(), Update{TMDbAPIKey: "private-key", TMDbLanguage: "zh-CN"}); err != nil {
		t.Fatal(err)
	}
	view, err := service.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !view.TMDbAPIKeyConfigured || view.TMDbLanguage != "zh-CN" {
		t.Fatalf("unexpected settings view: %#v", view)
	}
}
