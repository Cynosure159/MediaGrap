package database

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_migrate.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := t.Context()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify schema_migrations table exists and has entries
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Fatal("expected migrations to be recorded in schema_migrations")
	}

	// Idempotency check: running Migrate again should succeed without applying anything new
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate failed: %v", err)
	}

	var count2 int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	if err != nil || count2 != count {
		t.Fatalf("expected same migration count %d, got %d (err: %v)", count, count2, err)
	}
}
