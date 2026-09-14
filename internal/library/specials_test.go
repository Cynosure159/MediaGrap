package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/metadata"
)

func TestScanSpecialsBelongToParentShow(t *testing.T) {
	for _, folder := range []string{"Specials", "specials", "SpEcIaLs"} {
		for _, specialsOnly := range []bool{false, true} {
			name := folder + "/with-seasons"
			if specialsOnly {
				name = folder + "/specials-only"
			}
			t.Run(name, func(t *testing.T) {
				s, source, root, _ := scanBatchFixture(t, 0)
				showPath := "Example Show (2024)"
				specialPath := filepath.Join(showPath, folder, "Different.Filename.S00E02.mkv")
				snapshotFixture(t, root, specialPath)
				episodeCount, seasonCount := 1, 1
				if !specialsOnly {
					snapshotFixture(t, root, filepath.Join(showPath, "Season 01", "Example.S01E01.mkv"))
					snapshotFixture(t, root, filepath.Join(showPath, "Season 00", "Example.S00E01.mkv"))
					episodeCount, seasonCount = 3, 2
				}
				nfoRelative := filepath.Join(showPath, folder, "Different.Filename.S00E02.nfo")
				nfoBody := "<episodedetails><title>Local Special</title><season>0</season><episode>2</episode></episodedetails>"
				if err := os.WriteFile(filepath.Join(root, nfoRelative), []byte(nfoBody), 0600); err != nil {
					t.Fatal(err)
				}
				var showID, seasonID int64
				for _, mode := range []string{"incremental", "incremental", "full"} {
					if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
						t.Fatal(err)
					}
					shows, err := s.ListTVShows(t.Context(), "")
					if err != nil || len(shows) != 1 || shows[0].RelativePath != showPath || shows[0].TitleHint != "Example Show" || shows[0].YearHint == nil || *shows[0].YearHint != 2024 || shows[0].EpisodeCount != episodeCount || shows[0].SeasonCount != seasonCount {
						t.Fatalf("%s: shows=%+v err=%v", mode, shows, err)
					}
					if showID != 0 && shows[0].ID != showID {
						t.Fatal("show identity changed on rescan")
					}
					showID = shows[0].ID
					detail, err := s.TVShow(t.Context(), showID)
					if err != nil || len(detail.Episodes) != episodeCount {
						t.Fatalf("detail=%+v err=%v", detail, err)
					}
					found := false
					for _, episode := range detail.Episodes {
						if episode.RelativePath != specialPath {
							continue
						}
						found = true
						if episode.SeasonNumber != 0 || episode.EpisodeStart != 2 || episode.EpisodeEnd != 2 || len(episode.Sidecars) != 1 || episode.Sidecars[0].RelativePath != nfoRelative {
							t.Fatalf("special identity/sidecars=%+v", episode)
						}
					}
					if !found {
						t.Fatal("special absent from parent detail")
					}
					var currentSeasonID int64
					if err := s.db.QueryRow(`SELECT id FROM tv_seasons WHERE show_id=? AND season_number=0`, showID).Scan(&currentSeasonID); err != nil {
						t.Fatal(err)
					}
					if seasonID != 0 && currentSeasonID != seasonID {
						t.Fatal("season zero identity changed on rescan")
					}
					seasonID = currentSeasonID
					movies, err := s.ListMedia(t.Context(), "", 1, 50)
					if err != nil || movies.Total != 0 {
						t.Fatalf("episodes leaked into movies: %+v %v", movies, err)
					}
				}
				// Existing episode NFO stays untouched and retains explicit season zero.
				contents, err := os.ReadFile(filepath.Join(root, nfoRelative))
				if err != nil || string(contents) != nfoBody {
					t.Fatalf("NFO changed: %q %v", contents, err)
				}
				record, found, err := metadata.NewService(s.db, nil).ReadExistingTVNFO(showID, []metadata.TVNFOInput{{Kind: "episode", TargetPath: filepath.Join(root, nfoRelative)}})
				if err != nil || !found || len(record.Episodes) != 1 || record.Episodes[0].SeasonNumber != 0 || record.Episodes[0].EpisodeNumber != 2 || record.Episodes[0].Title != "Local Special" {
					t.Fatalf("NFO identity=%+v found=%t err=%v", record, found, err)
				}
			})
		}
	}
}

func TestScanSpecialsClassificationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		path, showPath string
		season         int
	}{
		{"Specials/Unrelated.S01E01.mkv", "Specials", 1},
		{"specials/Unrelated.S01E01.mkv", "specials", 1},
		{"Specials/Season 01/Unrelated.S01E01.mkv", "Specials", 1},
		{"Show/Special/Show.S00E01.mkv", "Show/Special", 0},
		{"Show/Specials Edition/Show.S00E01.mkv", "Show/Specials Edition", 0},
		{"Show/Bonus/Show.S00E01.mkv", "Show/Bonus", 0},
		{"Show/Extras/Show.S00E01.mkv", "Show/Extras", 0},
		{"Show/Specials/Show.S02E03.mkv", "Show", 2},
		{"Show/Specials/Bonus.mkv", "", 0},
		{"Show/Specials/Show.S00E01-sample.mkv", "", 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			s, source, root, _ := scanBatchFixture(t, 0)
			snapshotFixture(t, root, tc.path)
			if err := s.scan(t.Context(), 0, source.ID, "full", nil); err != nil {
				t.Fatal(err)
			}
			shows, err := s.ListTVShows(t.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			if tc.showPath == "" {
				if len(shows) != 0 {
					t.Fatalf("invented episode identity: %+v", shows)
				}
				movies, err := s.ListMedia(t.Context(), "", 1, 50)
				want := 1
				if isSampleVideo(filepath.Base(tc.path)) {
					want = 0
				}
				if err != nil || movies.Total != want {
					t.Fatalf("movie/sample classification: %+v %v", movies, err)
				}
				return
			}
			if len(shows) != 1 || shows[0].RelativePath != tc.showPath {
				t.Fatalf("show path: %+v want=%q", shows, tc.showPath)
			}
			detail, err := s.TVShow(t.Context(), shows[0].ID)
			if err != nil || len(detail.Episodes) != 1 || detail.Episodes[0].SeasonNumber != tc.season {
				t.Fatalf("explicit season changed: %+v %v", detail, err)
			}
		})
	}
}

func TestScanSpecialsFullRepairsLegacyGrouping(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 0)
	snapshotFixture(t, root, "Show/Season 01/Show.S01E01.mkv")
	snapshotFixture(t, root, "Show/Specials/Show.S00E01.mkv")
	if err := s.scan(t.Context(), 0, source.ID, "full", nil); err != nil {
		t.Fatal(err)
	}
	// Model the old classifier's separate show, retaining the current fingerprints.
	if _, err := s.db.Exec(`INSERT INTO tv_shows(source_id,relative_path,title_hint) VALUES(?,'Show/Specials','Specials');
		INSERT INTO tv_seasons(show_id,season_number) SELECT id,0 FROM tv_shows WHERE relative_path='Show/Specials';
		UPDATE tv_episodes SET show_id=(SELECT id FROM tv_shows WHERE relative_path='Show/Specials'),
		season_id=(SELECT id FROM tv_seasons WHERE show_id=(SELECT id FROM tv_shows WHERE relative_path='Show/Specials')) WHERE season_number=0;
		INSERT INTO tv_metadata(show_id,provider,title) SELECT id,'tmdb','Saved ' || title_hint FROM tv_shows;
		INSERT INTO tv_episode_metadata(show_id,season_number,episode_number,title) SELECT id,0,1,'Saved Special' FROM tv_shows WHERE relative_path='Show'`, source.ID); err != nil {
		t.Fatal(err)
	}
	var parentID, seasonID int64
	if err := s.db.QueryRow(`SELECT show_id,id FROM tv_seasons WHERE season_number=0 AND show_id=(SELECT id FROM tv_shows WHERE relative_path='Show')`).Scan(&parentID, &seasonID); err != nil {
		t.Fatal(err)
	}
	for scanIndex, mode := range []string{"incremental", "full", "incremental"} {
		if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
			t.Fatal(err)
		}
		shows, err := s.ListTVShows(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		if scanIndex == 0 {
			// The first warm scan deliberately preserves fingerprinted classification.
			if len(shows) != 2 {
				t.Fatalf("warm scan unexpectedly reclassified legacy rows: %+v", shows)
			}
			continue
		}
		if len(shows) != 1 || shows[0].ID != parentID || shows[0].RelativePath != "Show" || shows[0].Title != "Saved Show" || shows[0].EpisodeCount != 2 || shows[0].SeasonCount != 2 {
			t.Fatalf("%s: legacy grouping not repaired: %+v", mode, shows)
		}
		var currentSeasonID int64
		if err := s.db.QueryRow(`SELECT season_id FROM tv_episodes WHERE season_number=0`).Scan(&currentSeasonID); err != nil || currentSeasonID != seasonID {
			t.Fatalf("parent season not reused: %d %v", currentSeasonID, err)
		}
	}
	record, err := metadata.NewService(s.db, nil).TVRecord(t.Context(), parentID)
	if err != nil || record.Title != "Saved Show" || record.Provider != "tmdb" || len(record.Episodes) != 1 || record.Episodes[0].Title != "Saved Special" || record.Episodes[0].SeasonNumber != 0 {
		t.Fatalf("saved parent metadata changed: %+v %v", record, err)
	}
	var legacyTitle string
	if err := s.db.QueryRow(`SELECT title FROM tv_metadata WHERE show_id=(SELECT id FROM tv_shows WHERE relative_path='Show/Specials')`).Scan(&legacyTitle); err != nil || legacyTitle != "Saved Specials" {
		t.Fatalf("legacy saved metadata erased: %q %v", legacyTitle, err)
	}
}
