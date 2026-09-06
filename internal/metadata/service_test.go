package metadata

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

type staticMovieProvider struct{ details Details }

func (p staticMovieProvider) SearchMovies(context.Context, string, *int) ([]Candidate, error) {
	return nil, nil
}
func (p staticMovieProvider) Movie(context.Context, string) (Details, error) { return p.details, nil }

type staticTVProvider struct {
	staticMovieProvider
	details  TVDetails
	episodes []TVEpisodeDetails
}

func (p staticTVProvider) SearchTV(context.Context, string, *int) ([]Candidate, error) {
	return nil, nil
}
func (p staticTVProvider) TV(context.Context, string) (TVDetails, error) { return p.details, nil }
func (p staticTVProvider) TVSeason(context.Context, string, int) ([]TVEpisodeDetails, error) {
	return p.episodes, nil
}

func TestSelectAndWriteReplacesMetadataAndWritesNFO(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	media := filepath.Join(root, "Example.mkv")
	if err := os.WriteFile(media, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	source, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	item, err := db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, filepath.Base(media), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := item.LastInsertId()
	year := 2024
	service := NewService(db, staticMovieProvider{details: Details{Candidate: Candidate{ID: "42", Title: "Selected title", OriginalTitle: "Original title", Year: &year}}})
	record, err := service.SelectAndWrite(ctx, itemID, "42", media, true, func(path string) bool { return strings.HasPrefix(path, root+string(filepath.Separator)) })
	if err != nil {
		t.Fatal(err)
	}
	if record.Title != "Selected title" || record.ProviderID != "42" {
		t.Fatalf("unexpected selected record: %#v", record)
	}
	content, err := os.ReadFile(strings.TrimSuffix(media, ".mkv") + ".nfo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<title>Selected title</title>") {
		t.Fatalf("NFO did not contain selected metadata: %s", content)
	}
}

func TestSelectTVAndWriteReplacesMetadataAndWritesAllNFOs(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	seasonDir := filepath.Join(root, "Example Show", "Season 01")
	if err := os.MkdirAll(seasonDir, 0o750); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(seasonDir, "Example.Show.S01E01.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	source, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "TV", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	show, err := db.ExecContext(ctx, `INSERT INTO tv_shows(source_id,relative_path,title_hint) VALUES(?,?,?)`, sourceID, "Example Show", "Example Show")
	if err != nil {
		t.Fatal(err)
	}
	showID, _ := show.LastInsertId()
	season, err := db.ExecContext(ctx, `INSERT INTO tv_seasons(show_id,season_number) VALUES(?,?)`, showID, 1)
	if err != nil {
		t.Fatal(err)
	}
	seasonID, _ := season.LastInsertId()
	media, err := db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, "Example Show/Season 01/Example.Show.S01E01.mkv", "Example Show", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	mediaID, _ := media.LastInsertId()
	if _, err := db.ExecContext(ctx, `INSERT INTO tv_episodes(media_item_id,show_id,season_id,season_number,episode_start,episode_end,title_hint) VALUES(?,?,?,?,?,?,?)`, mediaID, showID, seasonID, 1, 1, 1, "Example Show"); err != nil {
		t.Fatal(err)
	}
	provider := staticTVProvider{details: TVDetails{Candidate: Candidate{ID: "10", Title: "Selected Show"}}, episodes: []TVEpisodeDetails{{SeasonNumber: 1, EpisodeNumber: 1, Title: "Pilot"}}}
	service := NewService(db, provider)
	inputs := []TVNFOInput{
		{MediaItemID: mediaID, Kind: "show", TargetPath: filepath.Join(root, "Example Show", "tvshow.nfo")},
		{MediaItemID: mediaID, Kind: "season", SeasonNumber: 1, TargetPath: filepath.Join(seasonDir, "season.nfo")},
		{MediaItemID: mediaID, Kind: "episode", SeasonNumber: 1, EpisodeNumber: 1, TargetPath: strings.TrimSuffix(video, ".mkv") + ".nfo"},
	}
	if _, err := service.SelectTVAndWrite(ctx, showID, "10", []int{1}, inputs, true, func(path string) bool { return strings.HasPrefix(path, root+string(filepath.Separator)) }); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{inputs[0].TargetPath, inputs[1].TargetPath, inputs[2].TargetPath} {
		contents, readErr := os.ReadFile(target)
		if readErr != nil || len(contents) == 0 {
			t.Fatalf("expected written NFO at %q: %v", target, readErr)
		}
	}
}

func TestReadExistingTVNFOUsesShowAndEpisodeSidecars(t *testing.T) {
	root := t.TempDir()
	showNFO := filepath.Join(root, "tvshow.nfo")
	episodeNFO := filepath.Join(root, "Example.S01E01.nfo")
	if err := os.WriteFile(showNFO, []byte("<tvshow><title>Local Show</title><year>2024</year></tvshow>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodeNFO, []byte("<episodedetails><title>Local Pilot</title><season>1</season><episode>1</episode></episodedetails>"), 0o600); err != nil {
		t.Fatal(err)
	}
	record, found, err := NewService(nil, nil).ReadExistingTVNFO(7, []TVNFOInput{{Kind: "show", TargetPath: showNFO}, {Kind: "episode", TargetPath: episodeNFO}})
	if err != nil || !found || record.Title != "Local Show" || len(record.Episodes) != 1 || record.Episodes[0].Title != "Local Pilot" {
		t.Fatalf("unexpected record: %#v found=%t err=%v", record, found, err)
	}
}

func TestPreviewAndApplyReplacesExistingNFO(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	video := filepath.Join(root, "Example.2024.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, filepath.Base(video), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	year := 2024
	service := NewService(db, nil)
	plan, err := service.Preview(ctx, Record{MediaItemID: itemID, Title: "Example", Year: &year}, video, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Conflict {
		t.Fatal("new target should not conflict")
	}
	applied, err := service.Apply(ctx, plan.ID, func(path string) bool { return strings.HasPrefix(path, root+string(filepath.Separator)) })
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != "applied" {
		t.Fatalf("expected applied plan, got %s", applied.State)
	}
	content, err := os.ReadFile(strings.TrimSuffix(video, ".mkv") + ".nfo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<title>Example</title>") {
		t.Fatal("expected written NFO")
	}
	second, err := service.Preview(ctx, Record{MediaItemID: itemID, Title: "Changed"}, video, true)
	if err != nil {
		t.Fatal(err)
	}
	if !second.WillReplace {
		t.Fatal("existing NFO should be marked for replacement")
	}
	if _, err := service.Apply(ctx, second.ID, func(string) bool { return true }); err != nil {
		t.Fatalf("replace existing NFO: %v", err)
	}
	replaced, err := os.ReadFile(strings.TrimSuffix(video, ".mkv") + ".nfo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(replaced), "Changed") {
		t.Fatal("existing NFO was not replaced")
	}
}

func TestPreviewUsesExistingKodiMovieNFOAsTarget(t *testing.T) {
	root := t.TempDir()
	video := filepath.Join(root, "Example.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "movie.nfo"), []byte("<movie><title>Old</title></movie>"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(t.Context(), `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(t.Context(), `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, filepath.Base(video), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	plan, err := NewService(db, nil).Preview(t.Context(), Record{MediaItemID: itemID, Title: "New"}, video, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.TargetPath != filepath.Join(root, "movie.nfo") || !plan.WillReplace {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReadExistingNFOReturnsDraftWithoutPersisting(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "Existing.mkv")
	if err := os.WriteFile(media, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	nfo := strings.TrimSuffix(media, ".mkv") + ".nfo"
	if err := os.WriteFile(nfo, []byte(`<?xml version="1.0"?><movie><title>Existing title</title><year>2002</year><uniqueid type="tmdb">42</uniqueid></movie>`), 0o600); err != nil {
		t.Fatal(err)
	}
	record, found, err := NewService(nil, nil).ReadExistingNFO(media, 9)
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.Title != "Existing title" || record.ProviderID != "42" || record.Year == nil || *record.Year != 2002 {
		t.Fatalf("unexpected record: %#v", record)
	}
}

func TestReadExistingNFOAlwaysReturnsArrayFields(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "Sparse.mkv")
	if err := os.WriteFile(strings.TrimSuffix(media, ".mkv")+".nfo", []byte(`<movie><title>Sparse</title></movie>`), 0o600); err != nil {
		t.Fatal(err)
	}
	record, found, err := NewService(nil, nil).ReadExistingNFO(media, 1)
	if err != nil || !found {
		t.Fatalf("read NFO: found=%v err=%v", found, err)
	}
	if record.Genres == nil || record.LockedFields == nil {
		t.Fatalf("array fields must not be nil: %#v", record)
	}
}

func TestReadExistingNFOFallsBackToKodiMovieNFO(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "Release.Name.2023.mkv")
	if err := os.WriteFile(filepath.Join(root, "movie.nfo"), []byte(`<movie><title>Folder title</title></movie>`), 0o600); err != nil {
		t.Fatal(err)
	}
	record, found, err := NewService(nil, nil).ReadExistingNFO(media, 1)
	if err != nil || !found || record.Title != "Folder title" {
		t.Fatalf("expected movie.nfo fallback, record=%#v found=%v err=%v", record, found, err)
	}
}

func TestHydrateExistingNFOPersistsOnlyWhenSQLiteIsEmpty(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	video := filepath.Join(root, "Example.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "movie.nfo"), []byte("<movie><title>Local title</title></movie>"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	source, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	item, err := db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, "Example.mkv", "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := item.LastInsertId()
	service := NewService(db, nil)
	if err := service.HydrateExistingNFO(ctx, itemID, video); err != nil {
		t.Fatal(err)
	}
	record, err := service.Record(ctx, itemID)
	if err != nil || record.Title != "Local title" || record.Provider != "nfo" {
		t.Fatalf("unexpected hydrated record: %#v err=%v", record, err)
	}
	if _, err := service.Save(ctx, Record{MediaItemID: itemID, Provider: "tmdb", Title: "Selected title"}); err != nil {
		t.Fatal(err)
	}
	if err := service.HydrateExistingNFO(ctx, itemID, video); err != nil {
		t.Fatal(err)
	}
	record, err = service.Record(ctx, itemID)
	if err != nil || record.Title != "Selected title" || record.Provider != "tmdb" {
		t.Fatalf("existing SQLite metadata must be preserved: %#v err=%v", record, err)
	}
}

func TestPreviewAndApplyArtworkWritesKodiSidecars(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	media := filepath.Join(root, "Example.mkv")
	if err := os.WriteFile(media, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, filepath.Base(media), "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "image.tmdb.org" {
			t.Fatalf("unexpected artwork host: %s", request.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(bytes.NewReader([]byte{0xff, 0xd8, 0xff, 'j', 'p', 'e', 'g'}))}, nil
	})}
	service := NewService(db, NewTMDb(nil, client, "key"))
	plan, err := service.PreviewArtwork(ctx, Record{MediaItemID: itemID, Title: "Example", Provider: "tmdb", PosterURL: "https://image.tmdb.org/t/p/w500/poster.jpg", BackdropURL: "https://image.tmdb.org/t/p/w500/fanart.jpg"}, media, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Assets) != 2 || plan.Assets[0].TargetPath != filepath.Join(root, "poster.jpg") {
		t.Fatalf("unexpected artwork plan: %#v", plan)
	}
	applied, err := service.ApplyArtwork(ctx, plan.ID, func(path string) bool { return strings.HasPrefix(path, root+string(filepath.Separator)) })
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != "applied" {
		t.Fatalf("expected applied artwork plan, got %s", applied.State)
	}
	for _, filename := range []string{"poster.jpg", "fanart.jpg"} {
		contents, readErr := os.ReadFile(filepath.Join(root, filename))
		if readErr != nil || !bytes.Equal(contents, []byte{0xff, 0xd8, 0xff, 'j', 'p', 'e', 'g'}) {
			t.Fatalf("unexpected %s: %q, %v", filename, contents, readErr)
		}
	}
}

func TestPreviewArtworkRejectsNonTMDbURLs(t *testing.T) {
	_, err := NewService(nil, nil).PreviewArtwork(context.Background(), Record{Title: "Example", PosterURL: "https://example.test/image.jpg"}, "/media/Example.mkv", true)
	if err == nil || !strings.Contains(err.Error(), "image.tmdb.org") {
		t.Fatalf("expected TMDb URL validation error, got %v", err)
	}
}

func TestApplyTVArtworkWritesKodiSeasonTarget(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(t.TempDir(), "tv-artwork.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "TV", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(ctx, `INSERT INTO tv_shows(source_id,relative_path,title_hint) VALUES(?,?,?)`, sourceID, "Example Show", "Example Show")
	if err != nil {
		t.Fatal(err)
	}
	showID, _ := result.LastInsertId()
	showDirectory := filepath.Join(root, "Example Show")
	if err := os.MkdirAll(showDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(bytes.NewReader([]byte{0xff, 0xd8, 0xff, 't', 'v'}))}, nil
	})}
	service := NewService(db, NewTMDb(nil, client, "key"))
	season := 1
	candidate := TVArtworkCandidate{ID: "tv-season-poster", ShowID: showID, Scope: "season", SeasonNumber: &season, Provider: "fanart.tv", ProviderAssetID: "7", Kind: "season_poster", SourceURL: "https://assets.fanart.tv/fanart/tv/81189/seasonposter/example.jpg", PreviewURL: "https://assets.fanart.tv/preview/tv/81189/seasonposter/example.jpg", MimeType: "image/jpeg"}
	if err := service.repo.ReplaceTVArtworkCandidates(ctx, showID, "season", &season, []TVArtworkCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	plan, err := service.PreviewTVArtworkSelection(ctx, showID, "season", &season, []TVArtworkSelection{{Kind: "season_poster", CandidateID: candidate.ID}}, showDirectory, true)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := service.applyTVArtworkFiles(ctx, plan, func(int, string) {})
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != "applied" {
		t.Fatalf("unexpected plan state: %s", applied.State)
	}
	contents, err := os.ReadFile(filepath.Join(showDirectory, "season01-poster.jpg"))
	if err != nil || !bytes.Equal(contents, []byte{0xff, 0xd8, 0xff, 't', 'v'}) {
		t.Fatalf("unexpected TV artwork: %x, %v", contents, err)
	}
}

func TestOpenArtworkPreviewUsesServerArtworkClient(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(t.TempDir(), "preview.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	result, err := db.ExecContext(ctx, `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	result, err = db.ExecContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,?,?,?,?)`, sourceID, "Example.mkv", "Example", 5, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	previewURL := "https://assets.fanart.tv/preview/example-asset.png"
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != previewURL {
			t.Fatalf("unexpected preview URL: %s", request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("\x89PNG\r\n\x1a\n"))),
		}, nil
	})}
	service := NewService(db, NewTMDb(nil, client, "key"))
	if _, err := db.ExecContext(ctx, `INSERT INTO artwork_candidates(id,media_item_id,provider,provider_asset_id,kind,source_url,preview_url,mime_type) VALUES(?,?,?,?,?,?,?,?)`, "candidate-1", itemID, "fanart.tv", "asset-1", "poster", "https://assets.fanart.tv/fanart/example-asset.png", previewURL, "image/png"); err != nil {
		t.Fatal(err)
	}

	preview, err := service.OpenArtworkPreview(ctx, itemID, "candidate-1")
	if err != nil {
		t.Fatal(err)
	}
	defer preview.Body.Close()
	data, err := io.ReadAll(preview.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("unexpected preview bytes: %x", data)
	}
}
