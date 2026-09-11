package metadata

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/providers/fanart"
)

func TestFanartAcceptsTMDbIDImportedFromNFO(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source, err := db.Exec(`INSERT INTO sources(name,root_path) VALUES('Movies',?)`, root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	result, err := db.Exec(`INSERT INTO media_items(source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,'Blade.mkv','Blade',1,'now')`, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	path := filepath.Join(root, "Blade.mkv")
	if err = os.WriteFile(filepath.Join(root, "Blade.nfo"), []byte(`<movie><title>Blade Runner 2049</title><uniqueid type="tmdb">335984</uniqueid></movie>`), 0600); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, staticMovieProvider{})
	if err = service.HydrateExistingNFO(t.Context(), id, path); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := fanart.NewClient(nil, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/v3.2/movies/335984" {
			t.Errorf("wrong TMDb endpoint %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"movieposter":[{"id":"1","url":"https://assets.fanart.tv/fanart/movies/335984/movieposter/test.jpg","lang":"en"}]}`))}, nil
	})}, "fixture-key")
	service.SetFanartProvider(client)
	candidates, err := service.ArtworkCandidates(t.Context(), id)
	if err != nil || len(candidates) != 1 || calls != 1 {
		t.Fatalf("NFO artwork: %+v %v calls=%d", candidates, err, calls)
	}
	record, err := service.Record(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if record.Provider != "nfo" {
		t.Fatal("provenance changed")
	}
	for _, test := range []struct{ provider, id string }{{"nfo", ""}, {"nfo", "tt1856101"}, {"tmdb", "0"}, {"tmdb", "-1"}, {"tmdb", "../335984"}, {"imdb", "335984"}} {
		record.Provider = test.provider
		record.ProviderID = test.id
		if _, err = service.Save(t.Context(), record); err != nil {
			t.Fatal(err)
		}
		if _, err = service.ArtworkCandidates(t.Context(), id); err == nil {
			t.Fatalf("accepted invalid identity %+v", test)
		}
	}
	if calls != 1 {
		t.Fatal("invalid IDs reached provider")
	}
}
