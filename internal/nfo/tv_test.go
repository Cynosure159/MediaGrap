package nfo

import (
	"strings"
	"testing"
)

func TestTVShowAndEpisodeRoundTrip(t *testing.T) {
	year := 2023
	rating := 9.1
	votes := 50000
	show := TVShow{
		Title:         "Sample TV Show",
		OriginalTitle: "Original TV Show",
		Year:          &year,
		Plot:          "A great series.",
		Genres:        []string{"Action", "Thriller"},
		TMDbID:        "67890",
		PosterURL:     "https://image.tmdb.org/t/p/original/tv_poster.jpg",
		BackdropURL:   "https://image.tmdb.org/t/p/original/tv_backdrop.jpg",
		Rating:        &rating,
		Votes:         &votes,
		Status:        "Ended",
		Network:       "HBO",
		Cast:          []Person{{Name: "Actor TV", Role: "Main", Thumb: "https://image.tmdb.org/t/p/w185/cast.jpg"}},
	}

	showXML, err := WriteTVShow(show)
	if err != nil {
		t.Fatalf("WriteTVShow: %v", err)
	}
	parsedShow, err := ParseTVShow(showXML)
	if err != nil {
		t.Fatalf("ParseTVShow: %v", err)
	}
	if parsedShow.Title != show.Title || parsedShow.Network != "HBO" || parsedShow.Status != "Ended" || len(parsedShow.Cast) != 1 {
		t.Fatalf("show mismatch: %+v vs %+v", parsedShow, show)
	}

	// Episode
	runtime := 55
	ep := Episode{
		Title:    "Pilot",
		Plot:     "The start.",
		Season:   1,
		Episode:  1,
		AirDate:  "2023-01-15",
		Runtime:  &runtime,
		TMDbID:   "9999",
		StillURL: "https://image.tmdb.org/t/p/original/still.jpg",
	}
	epXML, err := WriteEpisode(ep)
	if err != nil {
		t.Fatalf("WriteEpisode: %v", err)
	}
	parsedEp, err := ParseEpisode(epXML)
	if err != nil {
		t.Fatalf("ParseEpisode: %v", err)
	}
	if parsedEp.Title != ep.Title || parsedEp.Season != 1 || parsedEp.Episode != 1 || parsedEp.AirDate != "2023-01-15" {
		t.Fatalf("episode mismatch: %+v vs %+v", parsedEp, ep)
	}

	// Season
	seasonXML, err := WriteSeason("Season 1", 1)
	if err != nil {
		t.Fatalf("WriteSeason: %v", err)
	}
	if !strings.Contains(string(seasonXML), "<title>Season 1</title>") || !strings.Contains(string(seasonXML), "<season>1</season>") {
		t.Fatalf("season xml mismatch: %s", string(seasonXML))
	}
}
