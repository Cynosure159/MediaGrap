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

func TestIntegrationUpgradePreservesExistingJobsWithoutBackfill(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version TEXT PRIMARY KEY,applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "0017" {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(body)); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(?,datetime('now'))`, entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO jobs(id,kind,state,payload) VALUES(42,'scan','succeeded','{"original":true}')`); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var payload, state string
	var version, count int
	if err = db.QueryRow(`SELECT payload,state,event_version FROM jobs WHERE id=42`).Scan(&payload, &state, &version); err != nil {
		t.Fatal(err)
	}
	if payload != `{"original":true}` || state != "succeeded" || version != 0 {
		t.Fatal("legacy job changed")
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM system_events`).Scan(&count); err != nil || count != 0 {
		t.Fatal("migration backfilled events")
	}
	if err = Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}

func TestScanFingerprintMigrationPreservesLegacyRows(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "scan-upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "0019" {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(?,datetime('now'))`, entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'legacy','/fixture'); INSERT INTO media_items(id,source_id,relative_path,title_hint,file_size,modified_at,last_seen_at) VALUES(1,1,'Movie.mkv','Legacy',7,'old','old'); INSERT INTO media_metadata(media_item_id,provider,title) VALUES(1,'nfo','Saved title')`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var title, fingerprint, nfoFingerprint, seen string
	if err := db.QueryRow(`SELECT meta.title,m.scan_fingerprint,m.nfo_fingerprint,m.last_seen_at FROM media_items m JOIN media_metadata meta ON meta.media_item_id=m.id WHERE m.id=1`).Scan(&title, &fingerprint, &nfoFingerprint, &seen); err != nil || title != "Saved title" || fingerprint != "" || nfoFingerprint != "" || seen != "old" {
		t.Fatalf("legacy row changed: %q %q %q %q %v", title, fingerprint, nfoFingerprint, seen, err)
	}
	if err := Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}
