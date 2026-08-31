package kodi

import (
	"strings"
	"testing"
)

func TestMovieRoundTripIsDeterministic(t *testing.T) {
	year, runtime := 1999, 136
	rating := 8.2
	votes := 12345
	input := Movie{Title: "The Matrix", OriginalTitle: "The Matrix", Year: &year, Plot: "A hacker learns the truth.", Runtime: &runtime, Genres: []string{"Science Fiction", "Action"}, TMDbID: "603", PosterURL: "https://image.test/poster.jpg", Rating: &rating, Votes: &votes, ContentRating: "R", Directors: []string{"Lana Wachowski", "Lilly Wachowski"}, Writers: []string{"Lana Wachowski"}, Studios: []string{"Village Roadshow"}, Cast: []Person{{Name: "Keanu Reeves", Role: "Neo", Thumb: "https://image.test/keanu.jpg"}}}
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
	for _, fragment := range []string{`<rating>8.2</rating>`, `<votes>12345</votes>`, `<mpaa>R</mpaa>`, `<director>Lana Wachowski</director>`, `<name>Keanu Reeves</name>`, `<role>Neo</role>`, `<thumb>https://image.test/keanu.jpg</thumb>`} {
		if !strings.Contains(string(first), fragment) {
			t.Fatalf("expected extended metadata %q in NFO: %s", fragment, first)
		}
	}
}
