package kodi

import (
	"strings"
	"testing"
)

func TestMovieRoundTripIsDeterministic(t *testing.T) {
	year, runtime := 1999, 136
	input := Movie{Title: "The Matrix", OriginalTitle: "The Matrix", Year: &year, Plot: "A hacker learns the truth.", Runtime: &runtime, Genres: []string{"Science Fiction", "Action"}, TMDbID: "603", PosterURL: "https://image.test/poster.jpg"}
	first, err := WriteMovie(input)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseMovie(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := WriteMovie(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("round trip changed output:\n%s\n---\n%s", first, second)
	}
	if !strings.Contains(string(first), `<uniqueid type="tmdb" default="true">603</uniqueid>`) {
		t.Fatal("TMDb ID missing")
	}
}
