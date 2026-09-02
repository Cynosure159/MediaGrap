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
	if err := os.WriteFile(filepath.Join(episodeDirectory, "Example.Show.S01E01E02.nfo"), []byte("<episodedetails/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Example Show (2024)", "poster.jpg"), []byte("image"), 0o600); err != nil {
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
	if err := service.scan(t.Context(), jobID, sourceID, "full", nil); err != nil {
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
	detail, err := service.TVShow(t.Context(), shows[0].ID)
	if err != nil || len(detail.Artwork) != 1 || detail.Artwork[0].Kind != "poster" || len(detail.Episodes[0].Sidecars) != 1 || detail.Episodes[0].Sidecars[0].Kind != "nfo" {
		t.Fatalf("unexpected TV detail: %#v err=%v", detail, err)
	}
}

func TestDirectoryLevelKodiSidecarsBelongToOnlyMovieInDirectory(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "Example Movie")
	if err := os.MkdirAll(movieDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "Example.Movie.mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"movie.nfo", "poster.jpg", "fanart.jpg", "disc.png", "keyart.webp"} {
		if err := os.WriteFile(filepath.Join(movieDir, name), []byte("sidecar"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	assets := currentSidecarsForMedia(root, "Example Movie/Example.Movie.mkv")
	if len(assets) != 5 {
		t.Fatalf("expected directory-level Kodi assets, got %#v", assets)
	}
	for _, expected := range []string{"Example Movie/movie.nfo", "Example Movie/poster.jpg", "Example Movie/fanart.jpg", "Example Movie/disc.png", "Example Movie/keyart.webp"} {
		found := false
		for _, asset := range assets {
			found = found || asset.RelativePath == expected
		}
		if !found {
			t.Fatalf("missing %q from %#v", expected, assets)
		}
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

func TestDeleteSourceRemovesOnlyIndexedRecords(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Movie.mkv")
	if err := os.WriteFile(mediaPath, []byte("video"), 0o600); err != nil {
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
	result, err := db.ExecContext(t.Context(), `INSERT INTO sources(name, root_path) VALUES(?, ?)`, "Media", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	if _, err := db.ExecContext(t.Context(), `INSERT INTO media_items(source_id, relative_path, title_hint, file_size, modified_at) VALUES(?, ?, ?, ?, ?)`, sourceID, "Movie.mkv", "Movie", 1, "2024-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	if err := NewService(db, []string{root}).DeleteSource(t.Context(), sourceID); err != nil {
		t.Fatal(err)
	}
	var sourceCount, mediaCount int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM sources`).Scan(&sourceCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM media_items`).Scan(&mediaCount); err != nil {
		t.Fatal(err)
	}
	if sourceCount != 0 || mediaCount != 0 {
		t.Fatalf("expected source and index records to be removed, got sources=%d media=%d", sourceCount, mediaCount)
	}
	if _, err := os.Stat(mediaPath); err != nil {
		t.Fatalf("source deletion must not affect mounted files: %v", err)
	}
}
