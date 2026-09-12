package library

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
		{"movie-to-episode", "Movie.2024", "Show.S01E01.New", "New", 1, 1},
		{"episode-to-movie", "Show.S01E01.Old", "Movie.2024", "", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, source, root, counts := scanBatchFixture(t, 0)
			s.SetFFprobePath("")
			snapshotFixture(t, root, tc.before+".mkv")
			scan := func() {
				t.Helper()
				if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
					t.Fatal(err)
				}
			}
			scan()
			var id int64
			if err := s.db.QueryRow(`SELECT id FROM media_items`).Scan(&id); err != nil {
				t.Fatal(err)
			}
			var plan RenamePlan
			var err error
			if tc.name == "movie-to-episode" {
				plan, err = s.PreviewRenamePlan(t.Context(), id, tc.after)
			} else {
				shows, e := s.ListTVShows(t.Context(), "")
				if e != nil || len(shows) != 1 {
					t.Fatalf("shows=%v err=%v", shows, e)
				}
				plan, err = s.PreviewTVRenamePlan(t.Context(), shows[0].ID, nil, nil, tc.after)
			}
			if err != nil || plan.HasConflicts {
				t.Fatalf("preview=%+v err=%v", plan, err)
			}
			if err = s.ApplyRenamePlan(t.Context(), plan.ID, nil); err != nil {
				t.Fatal(err)
			}
			var path, fingerprint string
			if err = s.db.QueryRow(`SELECT relative_path,scan_fingerprint FROM media_items WHERE id=?`, id).Scan(&path, &fingerprint); err != nil || path != tc.after+".mkv" || fingerprint != "" {
				t.Fatalf("path=%q fingerprint=%q err=%v", path, fingerprint, err)
			}
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil || string(data) != "fixture" {
				t.Fatalf("video=%q err=%v", data, err)
			}
			scan()
			var episodes int
			if err = s.db.QueryRow(`SELECT count(*) FROM tv_episodes WHERE media_item_id=?`, id).Scan(&episodes); err != nil {
				t.Fatal(err)
			}
			if tc.episode == 0 {
				if episodes != 0 {
					t.Fatal("stale TV association")
				}
			} else {
				var title string
				var season, episode int
				if err = s.db.QueryRow(`SELECT title_hint,season_number,episode_start FROM tv_episodes WHERE media_item_id=?`, id).Scan(&title, &season, &episode); err != nil || title != tc.title || season != tc.season || episode != tc.episode {
					t.Fatalf("episode=%q S%dE%d err=%v", title, season, episode, err)
				}
			}
			// Once reclassified, warm and sidecar-only scans must not rewrite TV rows.
			for _, sidecar := range []bool{false, true} {
				if sidecar {
					snapshotFixture(t, root, tc.after+".jpg")
				}
				counts.reset()
				scan()
				_, _, writes := counts.snapshot()
				if writes["tv_shows"] != 0 || writes["tv_seasons"] != 0 || writes["tv_episodes"] != 0 {
					t.Fatalf("unchanged TV rewritten: %v", writes)
				}
			}
		})
	}
}

func TestRenamePlanFailurePreservesScanFingerprint(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 0)
	s.SetFFprobePath("")
	snapshotFixture(t, root, "Show.S01E01.Old.mkv")
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	var id int64
	var before string
	if err := s.db.QueryRow(`SELECT id,scan_fingerprint FROM media_items`).Scan(&id, &before); err != nil || before == "" {
		t.Fatalf("fingerprint=%q err=%v", before, err)
	}
	plan, err := s.PreviewRenamePlan(t.Context(), id, "Show.S01E01.New")
	if err != nil || plan.HasConflicts {
		t.Fatalf("preview=%+v err=%v", plan, err)
	}
	snapshotFixture(t, root, "Show.S01E01.New.mkv") // Conflict introduced after preview.
	if err = s.ApplyRenamePlan(t.Context(), plan.ID, nil); err != nil {
		t.Fatal(err)
	}
	current, err := s.GetRenamePlan(t.Context(), plan.ID)
	if err != nil || current.State != "partial" {
		t.Fatalf("plan=%+v err=%v", current, err)
	}
	var path, after string
	if err = s.db.QueryRow(`SELECT relative_path,scan_fingerprint FROM media_items WHERE id=?`, id).Scan(&path, &after); err != nil || path != "Show.S01E01.Old.mkv" || after != before {
		t.Fatalf("path=%q fingerprint=%q err=%v", path, after, err)
	}
	for _, p := range []string{"Show.S01E01.Old.mkv", "Show.S01E01.New.mkv"} {
		data, err := os.ReadFile(filepath.Join(root, p))
		if err != nil || string(data) != "fixture" {
			t.Fatalf("file=%q data=%q err=%v", p, data, err)
		}
	}
}
