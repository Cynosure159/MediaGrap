package tmdb

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchMoviesAndDetails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/movie":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":550,"title":"Fight Club","original_title":"Fight Club","release_date":"1999-10-15","overview":"An insomniac office worker...","poster_path":"/poster.jpg"}]}`))
		case "/movie/550":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":550,"title":"Fight Club","original_title":"Fight Club","release_date":"1999-10-15","overview":"An insomniac office worker...","runtime":139,"vote_average":8.4,"vote_count":26000,"genres":[{"name":"Drama"}],"production_companies":[{"name":"Fox 2000 Pictures"}],"credits":{"cast":[{"name":"Brad Pitt","character":"Tyler Durden","profile_path":"/brad.jpg"}],"crew":[{"name":"David Fincher","job":"Director"}]},"release_dates":{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"R"}]}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClient(slog.Default(), ts.Client(), "mock-key")
	client.SetEndpoint(ts.URL)

	candidates, err := client.SearchMovies(t.Context(), "Fight Club", nil)
	if err != nil {
		t.Fatalf("SearchMovies failed: %v", err)
	}
	if len(candidates) != 1 || candidates[0].ID != "550" || candidates[0].Title != "Fight Club" || *candidates[0].Year != 1999 {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}

	details, err := client.Movie(t.Context(), "550")
	if err != nil {
		t.Fatalf("Movie details failed: %v", err)
	}
	if details.Title != "Fight Club" || len(details.Directors) != 1 || details.Directors[0] != "David Fincher" || details.ContentRating != "R" {
		t.Fatalf("unexpected details: %+v", details)
	}
}

func TestSearchTVAndDetails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/tv":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":1396,"name":"Breaking Bad","original_name":"Breaking Bad","first_air_date":"2008-01-20","overview":"A chemistry teacher...","poster_path":"/bb.jpg"}]}`))
		case "/tv/1396":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1396,"name":"Breaking Bad","original_name":"Breaking Bad","first_air_date":"2008-01-20","overview":"A chemistry teacher...","status":"Ended","vote_average":9.5,"vote_count":12000,"genres":[{"name":"Crime"}],"networks":[{"name":"AMC"}],"credits":{"cast":[{"name":"Bryan Cranston","character":"Walter White","profile_path":"/walter.jpg"}]},"seasons":[{"season_number":1}]}`))
		case "/tv/1396/season/1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"episodes":[{"episode_number":1,"name":"Pilot","overview":"Walter White turns to crime...","air_date":"2008-01-20","runtime":58,"still_path":"/pilot.jpg"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClient(slog.Default(), ts.Client(), "mock-key")
	client.SetEndpoint(ts.URL)

	candidates, err := client.SearchTV(t.Context(), "Breaking Bad", nil)
	if err != nil {
		t.Fatalf("SearchTV failed: %v", err)
	}
	if len(candidates) != 1 || candidates[0].ID != "1396" {
		t.Fatalf("unexpected TV candidates: %+v", candidates)
	}

	details, err := client.TV(t.Context(), "1396")
	if err != nil {
		t.Fatalf("TV details failed: %v", err)
	}
	if details.Title != "Breaking Bad" || details.Network != "AMC" || len(details.Cast) != 1 {
		t.Fatalf("unexpected TV details: %+v", details)
	}

	episodes, err := client.TVSeason(t.Context(), "1396", 1)
	if err != nil {
		t.Fatalf("TVSeason failed: %v", err)
	}
	if len(episodes) != 1 || episodes[0].Title != "Pilot" || *episodes[0].RuntimeMinutes != 58 {
		t.Fatalf("unexpected episodes: %+v", episodes)
	}
}

func TestOutboundProxyHonorsNoProxyAndSupportsSOCKS5(t *testing.T) {
	client, err := NewOutboundClient("http://proxy.local:7890", "localhost,.lan")
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	bypassed, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "https://nas.lan/status", nil))
	if err != nil || bypassed != nil {
		t.Fatalf("expected NO_PROXY bypass, proxy=%v err=%v", bypassed, err)
	}
	proxied, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "https://api.themoviedb.org/3/configuration", nil))
	if err != nil || proxied == nil || proxied.Host != "proxy.local:7890" {
		t.Fatalf("expected HTTP proxy, proxy=%v err=%v", proxied, err)
	}
	if _, err := NewOutboundClient("socks5://user:pass@proxy.local:1080", "localhost"); err != nil {
		t.Fatalf("SOCKS5 proxy rejected: %v", err)
	}
	if _, err := NewOutboundClient("ftp://proxy.local:21"); err == nil {
		t.Fatal("expected unsupported proxy scheme to fail")
	}
}
