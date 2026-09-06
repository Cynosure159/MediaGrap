package fanart

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMovieMapsGroupsAndSortsPreferredLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-key") != "project-key" {
			t.Fatalf("missing API key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"movieposter":[{"id":"en","url":"https://assets.fanart.tv/poster-en.jpg","lang":"en","likes":99,"width":1000,"height":1500},{"id":"zh","url":"https://assets.fanart.tv/poster-zh.jpg","lang":"zh-CN","likes":1,"width":1000,"height":1500}],"hdmovielogo":[{"id":"logo","url":"https://assets.fanart.tv/logo.png","lang":"00","likes":4,"width":800,"height":310}]}`))
	}))
	defer server.Close()
	client := NewClient(slog.Default(), server.Client(), "project-key")
	client.SetEndpoint(server.URL)
	client.Configure(server.Client(), "project-key", "zh-CN")
	assets, err := client.Movie(t.Context(), "550")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 3 || assets[0].ID != "zh" || assets[0].Kind != "poster" || assets[2].Kind != "clearlogo" {
		t.Fatalf("unexpected assets: %#v", assets)
	}
}

func TestTVMapsShowAndSeasonArtwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv/81189" || r.Header.Get("client-key") != "personal-key" {
			t.Fatalf("unexpected TV request: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tvposter":[{"id":"poster","url":"https://assets.fanart.tv/fanart/tv/81189/tvposter/poster.jpg","lang":"en","likes":"9"}],"seasonposter":[{"id":"season","url":"https://assets.fanart.tv/fanart/tv/81189/seasonposter/season.jpg","lang":"zh","season":"1","likes":"4","width":"1000","height":"1426"}]}`))
	}))
	defer server.Close()
	client := NewClient(slog.Default(), server.Client(), "project-key")
	client.SetEndpoint(server.URL)
	client.ConfigureKeys(server.Client(), "project-key", "personal-key", "zh-CN")
	assets, err := client.TV(t.Context(), "81189")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 || assets[0].Kind != "poster" || assets[1].Kind != "season_poster" || assets[1].Season != "1" || assets[1].Width != 1000 {
		t.Fatalf("unexpected TV artwork: %#v", assets)
	}
}

func TestPersonalKeyDoesNotReplaceProjectAPIKey(t *testing.T) {
	client := NewClient(slog.Default(), http.DefaultClient, "")
	client.ConfigureKeys(http.DefaultClient, "", "personal-key", "en")
	if _, err := client.TV(t.Context(), "81189"); err == nil {
		t.Fatal("expected missing project API key error")
	}
}
