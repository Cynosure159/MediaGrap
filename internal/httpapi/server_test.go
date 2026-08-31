package httpapi_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
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

func TestSaveMetadataWritesOneJSONDocument(t *testing.T) {
	db := testDatabase(t)
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Example.2024.mkv")
	if err := os.WriteFile(mediaPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := db.ExecContext(t.Context(), `INSERT INTO sources(name, root_path) VALUES(?, ?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	item, err := db.ExecContext(t.Context(), `INSERT INTO media_items(source_id, relative_path, title_hint, file_size, modified_at) VALUES(?, ?, ?, ?, ?)`, sourceID, filepath.Base(mediaPath), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := item.LastInsertId()

	authService := auth.NewService(db)
	session, err := authService.Setup(t.Context(), "admin", "a-strong-test-password")
	if err != nil {
		t.Fatal(err)
	}
	server := httpapi.NewServer(slog.Default(), db, httpapi.BuildInfo{Version: "test", Commit: "abc"}, authService, library.NewService(db, []string{root}), metadata.NewService(db, nil))
	body := []byte(`{"title":"Example","genres":[]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/media/"+strconv.FormatInt(itemID, 10)+"/metadata", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", session.CSRFToken)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: session.Token})
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	decoder := json.NewDecoder(response.Body)
	var record metadata.Record
	if err := decoder.Decode(&record); err != nil {
		t.Fatalf("decode metadata response: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("expected exactly one JSON document, got trailing response: %v", err)
	}
}

func testServer(db *sql.DB) http.Handler {
	return httpapi.NewServer(slog.Default(), db, httpapi.BuildInfo{Version: "test", Commit: "abc"}, auth.NewService(db), library.NewService(db, []string{tTempRootPlaceholder}), metadata.NewService(db, metadata.NewTMDb(slog.Default(), http.DefaultClient, "")))
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
