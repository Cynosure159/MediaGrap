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
