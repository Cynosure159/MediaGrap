package metadata

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type tmdbRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn tmdbRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestMovieMapsExtendedDetails(t *testing.T) {
	client := &http.Client{Transport: tmdbRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/3/movie/123" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("append_to_response") != "credits,release_dates" {
			t.Fatalf("extended fields were not requested: %s", request.URL.RawQuery)
		}
		body := `{"id":123,"title":"Example","original_title":"Example","release_date":"2024-01-01","runtime":120,"vote_average":7.8,"vote_count":456,"genres":[{"name":"Drama"}],"production_companies":[{"name":"Example Studio"}],"credits":{"cast":[{"name":"Actor One","character":"Lead","profile_path":"/actor.jpg"}],"crew":[{"name":"Director One","job":"Director"},{"name":"Writer One","job":"Screenplay"}]},"release_dates":{"results":[{"iso_3166_1":"CN","release_dates":[{"certification":"PG-13"}]}]}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	provider := NewTMDb(nil, client, "test-key")
	provider.Configure(client, "test-key", "zh-CN")

	details, err := provider.Movie(t.Context(), "123")
	if err != nil {
		t.Fatalf("movie details: %v", err)
	}
	if details.Rating == nil || *details.Rating != 7.8 || details.Votes == nil || *details.Votes != 456 {
		t.Fatalf("rating was not mapped: %#v", details)
	}
	if details.ContentRating != "PG-13" || len(details.Directors) != 1 || len(details.Writers) != 1 || len(details.Studios) != 1 {
		t.Fatalf("extended credits were not mapped: %#v", details)
	}
	if len(details.Cast) != 1 || details.Cast[0].Role != "Lead" || !strings.Contains(details.Cast[0].ProfileURL, "/actor.jpg") {
		t.Fatalf("cast was not mapped: %#v", details.Cast)
	}
}

func TestTVMapsDetailsAndSeasonEpisodes(t *testing.T) {
	client := &http.Client{Transport: tmdbRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body string
		switch request.URL.Path {
		case "/3/tv/42":
			if request.URL.Query().Get("append_to_response") != "credits" {
				t.Fatalf("credits were not requested: %s", request.URL.RawQuery)
			}
			body = `{"id":42,"name":"Example Show","original_name":"Example Show","first_air_date":"2023-01-01","vote_average":8.1,"vote_count":99,"genres":[{"name":"Drama"}],"networks":[{"name":"Example Network"}],"credits":{"cast":[{"name":"Actor","character":"Lead","profile_path":"/actor.jpg"}]}}`
		case "/3/tv/42/season/1":
			body = `{"episodes":[{"episode_number":2,"name":"Episode Two","overview":"Remote episode","air_date":"2023-01-08","runtime":48,"still_path":"/still.jpg"}]}`
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	provider := NewTMDb(nil, client, "test-key")
	details, err := provider.TV(t.Context(), "42")
	if err != nil {
		t.Fatalf("TV details: %v", err)
	}
	if details.Rating == nil || *details.Rating != 8.1 || details.Network != "Example Network" || len(details.Cast) != 1 {
		t.Fatalf("TV details were not mapped: %#v", details)
	}
	episodes, err := provider.TVSeason(t.Context(), "42", 1)
	if err != nil {
		t.Fatalf("season details: %v", err)
	}
	if len(episodes) != 1 || episodes[0].EpisodeNumber != 2 || episodes[0].RuntimeMinutes == nil || *episodes[0].RuntimeMinutes != 48 {
		t.Fatalf("episode details were not mapped: %#v", episodes)
	}
}

func TestMovieUsesConfiguredFallbackLanguageForMissingTranslation(t *testing.T) {
	languages := []string{}
	client := &http.Client{Transport: tmdbRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		language := request.URL.Query().Get("language")
		languages = append(languages, language)
		body := `{"id":123,"title":"","original_title":"Original","overview":""}`
		if language == "en-US" {
			body = `{"id":123,"title":"Fallback title","original_title":"Original","overview":"Fallback overview"}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	provider := NewTMDb(nil, client, "test-key")
	provider.ConfigureAdvanced(client, "test-key", "zh-CN", "en-US")
	details, err := provider.Movie(t.Context(), "123")
	if err != nil {
		t.Fatal(err)
	}
	if details.Title != "Fallback title" || details.Overview != "Fallback overview" {
		t.Fatalf("fallback translation not merged: %#v", details)
	}
	if len(languages) != 2 || languages[0] != "zh-CN" || languages[1] != "en-US" {
		t.Fatalf("unexpected language sequence: %#v", languages)
	}
}
