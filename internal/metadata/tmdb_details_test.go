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
