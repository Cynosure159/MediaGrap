package kodi

import (
	"strings"
	"testing"
)

func TestTVShowRoundTrip(t *testing.T) {
	year := 2024
	rating := 8.2
	content, err := WriteTVShow(TVShow{Title: "Example", OriginalTitle: "Example Original", Year: &year, Plot: "Plot", Genres: []string{"Drama"}, TMDbID: "42", Rating: &rating, Network: "Network", Cast: []Person{{Name: "Actor", Role: "Lead"}}})
	if err != nil {
		t.Fatal(err)
	}
	show, err := ParseTVShow(content)
	if err != nil {
		t.Fatal(err)
	}
	if show.Title != "Example" || show.TMDbID != "42" || show.Rating == nil || *show.Rating != 8.2 || len(show.Cast) != 1 {
		t.Fatalf("unexpected show: %#v", show)
	}
}

func TestEpisodeWriteIsKodiEpisodeDetails(t *testing.T) {
	content, err := WriteEpisode(Episode{Title: "Pilot", Season: 1, Episode: 2, TMDbID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "<episodedetails>") || !strings.Contains(string(content), "<season>1</season>") {
		t.Fatalf("unexpected XML: %s", content)
	}
	parsed, err := ParseEpisode(content)
	if err != nil || parsed.Title != "Pilot" || parsed.Season != 1 || parsed.Episode != 2 {
		t.Fatalf("unexpected episode: %#v err=%v", parsed, err)
	}
}
