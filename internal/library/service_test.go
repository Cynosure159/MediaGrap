package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestParseEpisodeHint(t *testing.T) {
	cases := []struct {
		filename            string
		season, first, last int
	}{
		{"Show.Name.S02E03.mkv", 2, 3, 3},
		{"Show Name - 1x02 - Pilot.mp4", 1, 2, 2},
		{"Show.S01E01E02.mkv", 1, 1, 2},
	}
	for _, test := range cases {
		season, first, last, ok := parseEpisodeHint(test.filename)
		if !ok || season != test.season || first != test.first || last != test.last {
			t.Fatalf("%q: got S%02dE%02d-%02d ok=%t", test.filename, season, first, last, ok)
		}
	}
}

func TestTVShowHintUsesDirectoryAboveSeason(t *testing.T) {
	path, title, year := tvShowHint("示例剧集 (2024)/Season 01/示例剧集.S01E02.mkv")
	if path != "示例剧集 (2024)" || title != "示例剧集" || year == nil || *year != 2024 {
		t.Fatalf("unexpected show hint: path=%q title=%q year=%v", path, title, year)
	}
}

func TestScanIndexesTVSeparatelyFromMovies(t *testing.T) {
	root := t.TempDir()
	episodeDirectory := filepath.Join(root, "Example Show (2024)", "Season 01")
	if err := os.MkdirAll(episodeDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(episodeDirectory, "Example.Show.S01E01E02.mkv"), []byte("episode"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Example.Movie.2024.mkv"), []byte("movie"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "mediagrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(t.Context(), `INSERT INTO sources(name, root_path) VALUES(?, ?)`, "Fixture", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(t.Context(), `INSERT INTO jobs(kind, source_id, state) VALUES('scan', ?, 'running')`, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	jobID, _ := result.LastInsertId()
	service := NewService(db, []string{root})
	if err := service.scan(t.Context(), jobID, sourceID); err != nil {
		t.Fatal(err)
	}
	shows, err := service.ListTVShows(t.Context(), "")
	if err != nil || len(shows) != 1 || shows[0].TitleHint != "Example Show" || shows[0].EpisodeCount != 1 || shows[0].SeasonCount != 1 {
		t.Fatalf("unexpected shows: %#v err=%v", shows, err)
	}
	movies, err := service.ListMedia(t.Context(), "", 1, 50)
	if err != nil || len(movies.Items) != 1 || movies.Items[0].RelativePath != "Example.Movie.2024.mkv" {
		t.Fatalf("unexpected movies: %#v err=%v", movies.Items, err)
	}
}

func TestMetadataSearchHintPrefersMovieDirectory(t *testing.T) {
	service := NewService(nil, nil)
	title, year, origin := service.MetadataSearchHint(MediaItem{
		RelativePath: "奥本海默 (2023)/Oppenheimer.2023.1080p.MA.WEB-DL.DUAL.DD+5.1.H.265-TheBiscuitMan.mkv",
		TitleHint:    "Oppenheimer 1080p MA WEB DL DUAL DD+5 1 H 265 TheBiscuitMan",
	})
	if title != "奥本海默" || year == nil || *year != 2023 || origin != "parent_directory" {
		t.Fatalf("unexpected directory search hint: title=%q year=%v origin=%q", title, year, origin)
	}
}

func TestMetadataSearchHintFallsBackToFilenameAtSourceRoot(t *testing.T) {
	service := NewService(nil, nil)
	year := 2010
	title, actualYear, origin := service.MetadataSearchHint(MediaItem{RelativePath: "Inception.2010.mkv", TitleHint: "Inception", YearHint: &year})
	if title != "Inception" || actualYear != &year || origin != "filename" {
		t.Fatalf("unexpected filename search hint: title=%q year=%v origin=%q", title, actualYear, origin)
	}
}
