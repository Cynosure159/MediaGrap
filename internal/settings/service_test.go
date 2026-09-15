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

func TestRenamePatternValidationCoversOptionalMovieAndTVGrammar(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern string
		tv      bool
		valid   bool
	}{
		{"movie optional unavailable", "${title}${ - ,edition,}", false, true},
		{"tv optional resolution", "${showTitle}${ [,resolution,]}", true, true},
		{"tv strict missing is runtime error but syntax is valid", "${seasonNumber}", true, true},
		{"unknown optional", "${showTitle}${,edition,}", true, false},
		{"extra comma", "${title,a,b,c}", false, false},
		{"dot affix", "${.,title,}", false, false},
		{"malformed literal", "${title}{", false, false},
		{"empty path after optional", "${title}/${,edition,}", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRenamePattern(tc.pattern, tc.tv)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
}

func TestRenameDefaultsPersistAndValidate(t *testing.T) {
	db, err := database.Open(t.TempDir() + "/mediagrap.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, Defaults{})
	view, err := service.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if view.MovieRenamePattern != DefaultMovieRenamePattern || view.TVRenamePattern != DefaultTVRenamePattern {
		t.Fatal("missing defaults")
	}
	movie, tv := "${title}/${title}", "${showTitle}/S${seasonNumberPad}E${episodeNumberPad}"
	if _, err := service.Update(t.Context(), Update{MovieRenamePattern: &movie, TVRenamePattern: &tv}); err != nil {
		t.Fatal(err)
	}
	optional := "${title}${ - ,edition,}"
	if _, err := service.Update(t.Context(), Update{MovieRenamePattern: &optional}); err != nil {
		t.Fatalf("optional pattern rejected: %v", err)
	}
	movie = optional
	if _, err := service.Update(t.Context(), Update{TMDbLanguage: "zh-CN"}); err != nil {
		t.Fatal(err)
	}
	view, err = NewService(db, Defaults{}).View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if view.MovieRenamePattern != movie || view.TVRenamePattern != tv {
		t.Fatal("patterns not preserved")
	}
	for _, invalid := range []string{"", "../movie", "/movie", "${unknown}", "${showTitle}", "${title", "a\\b", "a//b"} {
		if _, err := service.Update(t.Context(), Update{MovieRenamePattern: &invalid}); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
	view, _ = service.View(t.Context())
	if view.MovieRenamePattern != movie {
		t.Fatal("invalid update changed saved pattern")
	}
}
