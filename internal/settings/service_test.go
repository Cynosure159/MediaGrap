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

func TestUpdatePersistsNetworkAndPerUserPreferences(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/mediagrap.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(t.Context(), `INSERT INTO users(username,password_hash) VALUES('admin','hash')`)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := result.LastInsertId()
	service := NewService(db, Defaults{TMDbLanguage: "en-US", FallbackLanguage: "en-US"})
	_, err = service.Update(t.Context(), Update{FallbackLanguage: "zh-CN", OutboundProxy: "socks5://proxy.local:1080", NoProxy: "localhost,.lan", Theme: "system", Locale: "zh-CN"}, userID)
	if err != nil {
		t.Fatal(err)
	}
	view, err := service.View(t.Context(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if view.FallbackLanguage != "zh-CN" || !view.OutboundProxyConfigured || !view.NoProxyConfigured || view.Theme != "system" || view.Locale != "zh-CN" {
		t.Fatalf("unexpected settings view: %#v", view)
	}
}

func TestUpdateRejectsUnsafeNoProxyEntry(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/mediagrap.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, Defaults{})
	if _, err := service.Update(t.Context(), Update{NoProxy: "https://example.com/path"}); err == nil {
		t.Fatal("expected invalid NO_PROXY entry")
	}
}
