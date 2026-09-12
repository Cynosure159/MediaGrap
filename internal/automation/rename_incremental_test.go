package automation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamePlanIncrementalClassification(t *testing.T) {
	for _, tc := range []struct {
		name, before, after, title string
		season, episode            int
	}{
		{"episode-title", "Show.S01E01.Old", "Show.S01E01.New", "New", 1, 1},
		{"episode-tokens", "Show.S01E01.Old", "Show.S02E03.New", "New", 2, 3},
		{"movie-to-episode", "movie", "Show.S01E01.New", "New", 1, 1},
		{"episode-to-movie", "Show.S01E01.Old", "Movie.2024", "", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, root := fixture(t)
			if tc.before != "movie" {
				if err := os.Rename(filepath.Join(root, "movie.mkv"), filepath.Join(root, tc.before+".mkv")); err != nil {
					t.Fatal(err)
				}
				if _, err := s.db.Exec(`UPDATE media_items SET relative_path=? WHERE id=1`, tc.before+".mkv"); err != nil {
					t.Fatal(err)
				}
			}
			scan := func() {
				t.Helper()
				if err := s.library.ScanForAutomation(t.Context(), 0, 1, nil); err != nil {
					t.Fatal(err)
				}
			}
			scan()
			var before string
			if err := s.db.QueryRow(`SELECT scan_fingerprint FROM media_items WHERE id=1`).Scan(&before); err != nil || before == "" {
				t.Fatalf("fingerprint=%q err=%v", before, err)
			}
			plan, err := s.Preview(t.Context(), p, "rename", PreviewInput{MediaID: 1, Pattern: tc.after, IdempotencyKey: "rename-preview"})
			if err != nil {
				t.Fatal(err)
			}
			q := approveAndQueue(t, s, p, plan)
			if err = execute(t, s, q); err != nil {
				t.Fatal(err)
			}
			var path, fingerprint string
			if err = s.db.QueryRow(`SELECT relative_path,scan_fingerprint FROM media_items WHERE id=1`).Scan(&path, &fingerprint); err != nil || path != tc.after+".mkv" || fingerprint != "" {
				t.Fatalf("path=%q fingerprint=%q err=%v", path, fingerprint, err)
			}
			scan()
			var count int
			if err = s.db.QueryRow(`SELECT count(*) FROM tv_episodes WHERE media_item_id=1`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if tc.episode == 0 {
				if count != 0 {
					t.Fatal("stale TV association")
				}
			} else {
				var title string
				var season, episode int
				if err = s.db.QueryRow(`SELECT title_hint,season_number,episode_start FROM tv_episodes WHERE media_item_id=1`).Scan(&title, &season, &episode); err != nil || title != tc.title || season != tc.season || episode != tc.episode {
					t.Fatalf("episode=%q S%dE%d err=%v", title, season, episode, err)
				}
			}
			var saved string
			if err = s.db.QueryRow(`SELECT title FROM media_metadata WHERE media_item_id=1`).Scan(&saved); err != nil || saved != "Approved title" {
				t.Fatalf("saved metadata=%q err=%v", saved, err)
			}
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil || string(data) != "valuable video" {
				t.Fatalf("video=%q err=%v", data, err)
			}
		})
	}
}

func TestRenamePlanRollbackPreservesFingerprintAndRecoveryInvalidates(t *testing.T) {
	s, p, root := fixture(t)
	if err := s.library.ScanForAutomation(t.Context(), 0, 1, nil); err != nil {
		t.Fatal(err)
	}
	var before string
	if err := s.db.QueryRow(`SELECT scan_fingerprint FROM media_items WHERE id=1`).Scan(&before); err != nil || before == "" {
		t.Fatalf("fingerprint=%q err=%v", before, err)
	}
	plan, err := s.Preview(t.Context(), p, "rename", PreviewInput{MediaID: 1, Pattern: "Show.S02E03.New", IdempotencyKey: "rename-rollback"})
	if err != nil {
		t.Fatal(err)
	}
	q := approveAndQueue(t, s, p, plan)
	// Fail after the media UPDATE inside commitOperation, proving invalidation rolls back too.
	if _, err = s.db.Exec(`CREATE TRIGGER fail_rename_audit BEFORE INSERT ON audit_entries WHEN NEW.action='automation.rename' BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err = execute(t, s, q); err == nil {
		t.Fatal("expected injected failure")
	}
	var path, after string
	if err = s.db.QueryRow(`SELECT relative_path,scan_fingerprint FROM media_items WHERE id=1`).Scan(&path, &after); err != nil || path != "movie.mkv" || after != before {
		t.Fatalf("rollback path=%q fingerprint=%q err=%v", path, after, err)
	}
	data, err := os.ReadFile(filepath.Join(root, "Show.S02E03.New.mkv"))
	if err != nil || string(data) != "valuable video" {
		t.Fatalf("published video=%q err=%v", data, err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER fail_rename_audit`); err != nil {
		t.Fatal(err)
	}
	if err = s.Recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT relative_path,scan_fingerprint FROM media_items WHERE id=1`).Scan(&path, &after); err != nil || path != "Show.S02E03.New.mkv" || after != "" {
		t.Fatalf("recovery path=%q fingerprint=%q err=%v", path, after, err)
	}
	current, err := s.Plan(t.Context(), p, plan.ID)
	if err != nil || current.State != "applied" {
		t.Fatalf("plan=%+v err=%v", current, err)
	}
	if err = s.library.ScanForAutomation(t.Context(), 0, 1, nil); err != nil {
		t.Fatal(err)
	}
	var title string
	var season, episode int
	if err = s.db.QueryRow(`SELECT title_hint,season_number,episode_start FROM tv_episodes WHERE media_item_id=1`).Scan(&title, &season, &episode); err != nil || title != "New" || season != 2 || episode != 3 {
		t.Fatalf("recovered episode=%q S%dE%d err=%v", title, season, episode, err)
	}
	restored, err := os.ReadFile(filepath.Join(root, path))
	if err != nil || string(restored) != string(data) {
		t.Fatalf("recovery mutated video: %q %v", restored, err)
	}
}
