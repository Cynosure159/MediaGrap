package kodi

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// Movie is the intentionally small, Kodi-compatible metadata shape used by the MVP.
type Movie struct {
	Title         string
	OriginalTitle string
	Year          *int
	Plot          string
	Runtime       *int
	Genres        []string
	TMDbID        string
	PosterURL     string
	BackdropURL   string
	Rating        *float64
	Votes         *int
	ContentRating string
	Directors     []string
	Writers       []string
	Studios       []string
	Cast          []Person
}
type Person struct {
	Name  string
	Role  string
	Thumb string
}

type movieXML struct {
	XMLName       xml.Name   `xml:"movie"`
	Title         string     `xml:"title"`
	OriginalTitle string     `xml:"originaltitle,omitempty"`
	Year          *int       `xml:"year,omitempty"`
	Plot          string     `xml:"plot,omitempty"`
	Runtime       *int       `xml:"runtime,omitempty"`
	Rating        *float64   `xml:"rating,omitempty"`
	Votes         *int       `xml:"votes,omitempty"`
	ContentRating string     `xml:"mpaa,omitempty"`
	Genres        []string   `xml:"genre,omitempty"`
	Directors     []string   `xml:"director,omitempty"`
	Writers       []string   `xml:"credits,omitempty"`
	Studios       []string   `xml:"studio,omitempty"`
	Actors        []actorXML `xml:"actor,omitempty"`
	UniqueID      []idXML    `xml:"uniqueid"`
	Thumb         []thumbXML `xml:"thumb"`
}
type actorXML struct {
	Name  string `xml:"name"`
	Role  string `xml:"role,omitempty"`
	Thumb string `xml:"thumb,omitempty"`
}
type idXML struct {
	Type    string `xml:"type,attr"`
	Default string `xml:"default,attr"`
	Value   string `xml:",chardata"`
}
type thumbXML struct {
	Aspect string `xml:"aspect,attr"`
	Value  string `xml:",chardata"`
}

func ParseMovie(input []byte) (Movie, error) {
	var decoded movieXML
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = true
	if err := decoder.Decode(&decoded); err != nil {
		return Movie{}, err
	}
	movie := Movie{Title: strings.TrimSpace(decoded.Title), OriginalTitle: strings.TrimSpace(decoded.OriginalTitle), Year: decoded.Year, Plot: strings.TrimSpace(decoded.Plot), Runtime: decoded.Runtime, Genres: cleaned(decoded.Genres), Rating: decoded.Rating, Votes: decoded.Votes, ContentRating: strings.TrimSpace(decoded.ContentRating), Directors: cleaned(decoded.Directors), Writers: cleaned(decoded.Writers), Studios: cleaned(decoded.Studios)}
	for _, actor := range decoded.Actors {
		if name := strings.TrimSpace(actor.Name); name != "" {
			movie.Cast = append(movie.Cast, Person{Name: name, Role: strings.TrimSpace(actor.Role), Thumb: strings.TrimSpace(actor.Thumb)})
		}
	}
	for _, id := range decoded.UniqueID {
		if strings.EqualFold(id.Type, "tmdb") {
			movie.TMDbID = strings.TrimSpace(id.Value)
			break
		}
	}
	for _, thumb := range decoded.Thumb {
		switch strings.ToLower(thumb.Aspect) {
		case "poster":
			movie.PosterURL = strings.TrimSpace(thumb.Value)
		case "fanart":
			movie.BackdropURL = strings.TrimSpace(thumb.Value)
		}
	}
	return movie, nil
}

func WriteMovie(movie Movie) ([]byte, error) {
	encoded := movieXML{Title: strings.TrimSpace(movie.Title), OriginalTitle: strings.TrimSpace(movie.OriginalTitle), Year: movie.Year, Plot: strings.TrimSpace(movie.Plot), Runtime: movie.Runtime, Genres: cleaned(movie.Genres), Rating: movie.Rating, Votes: movie.Votes, ContentRating: strings.TrimSpace(movie.ContentRating), Directors: cleaned(movie.Directors), Writers: cleaned(movie.Writers), Studios: cleaned(movie.Studios)}
	for _, person := range movie.Cast {
		if name := strings.TrimSpace(person.Name); name != "" {
			encoded.Actors = append(encoded.Actors, actorXML{Name: name, Role: strings.TrimSpace(person.Role), Thumb: strings.TrimSpace(person.Thumb)})
		}
	}
	if movie.TMDbID != "" {
		encoded.UniqueID = []idXML{{Type: "tmdb", Default: "true", Value: movie.TMDbID}}
	}
	if movie.PosterURL != "" {
		encoded.Thumb = append(encoded.Thumb, thumbXML{Aspect: "poster", Value: movie.PosterURL})
	}
	if movie.BackdropURL != "" {
		encoded.Thumb = append(encoded.Thumb, thumbXML{Aspect: "fanart", Value: movie.BackdropURL})
	}
	body, err := xml.MarshalIndent(encoded, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(body, '\n')...), nil
}

func cleaned(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
