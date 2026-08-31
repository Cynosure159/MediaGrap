package httpapi_test

import (
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestHealthEndpoint(t *testing.T) {
	db := testDatabase(t)
	server := testServer(db)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestSystemInfoEndpoint(t *testing.T) {
	db := testDatabase(t)
	server := testServer(db)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type %q", contentType)
	}
}

func testServer(db *sql.DB) http.Handler {
	return httpapi.NewServer(slog.Default(), db, httpapi.BuildInfo{Version: "test", Commit: "abc"}, auth.NewService(db), library.NewService(db, []string{tTempRootPlaceholder}))
}

const tTempRootPlaceholder = "/media"

func testDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/mediagrap.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
