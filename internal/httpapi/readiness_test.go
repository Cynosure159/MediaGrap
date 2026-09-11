package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestReadinessDetectsPoolExhaustionAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	s := &server{db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	rec := httptest.NewRecorder()
	start := time.Now()
	s.ready(rec, httptest.NewRequest("GET", "/readyz", nil).WithContext(ctx))
	if rec.Code != 503 || time.Since(start) > time.Second {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
	var body struct {
		Database database.PoolStats `json:"database"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Database.Saturated || body.Database.InUse != 1 || body.Database.MaxOpenConnections != 1 || body.Database.WaitCount == 0 {
		t.Fatalf("bad stats: %+v", body)
	}
	independent, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	independent.Close()
	conn.Close()
	rec = httptest.NewRecorder()
	s.ready(rec, httptest.NewRequest("GET", "/readyz", nil))
	if rec.Code != 200 {
		t.Fatalf("did not recover: %s", rec.Body)
	}
}

// An exhausted pool must not block the liveness handler, even with no DB.
func TestLivenessDoesNotAcquireDatabase(t *testing.T) {
	s := &server{db: (*sql.DB)(nil)}
	rec := httptest.NewRecorder()
	s.health(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
}
