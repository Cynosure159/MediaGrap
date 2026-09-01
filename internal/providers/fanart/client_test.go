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
