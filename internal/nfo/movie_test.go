package nfo

import (
	"strings"
	"testing"
)

func TestMovieRoundTrip(t *testing.T) {
	year := 2024
	runtime := 125
	rating := 8.3
	votes := 14200
	movie := Movie{
		Title:         "Sample Movie",
		OriginalTitle: "Original Sample Movie",
		Year:          &year,
		Plot:          "A gripping plot description.",
		Runtime:       &runtime,
		Genres:        []string{"Drama", "Sci-Fi"},
		TMDbID:        "12345",
		PosterURL:     "https://image.tmdb.org/t/p/original/poster.jpg",
		BackdropURL:   "https://image.tmdb.org/t/p/original/backdrop.jpg",
		Rating:        &rating,
		Votes:         &votes,
		ContentRating: "PG-13",
		Directors:     []string{"Director One"},
		Writers:       []string{"Writer One"},
		Studios:       []string{"Studio One"},
		Cast:          []Person{{Name: "Actor One", Role: "Hero", Thumb: "https://image.tmdb.org/t/p/w185/actor.jpg"}},
	}

	payload, err := WriteMovie(movie)
	if err != nil {
		t.Fatalf("WriteMovie: %v", err)
	}
	if !strings.HasPrefix(string(payload), "<?xml") {
		t.Fatalf("missing xml header: %s", string(payload))
	}

	parsed, err := ParseMovie(payload)
	if err != nil {
		t.Fatalf("ParseMovie: %v", err)
	}

	if parsed.Title != movie.Title || parsed.TMDbID != movie.TMDbID || *parsed.Year != *movie.Year || *parsed.Runtime != *movie.Runtime || *parsed.Rating != *movie.Rating || len(parsed.Cast) != 1 {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", parsed, movie)
	}
}
