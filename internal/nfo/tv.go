package nfo

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// TVShow and Episode represent the Kodi-compatible fields MediaGrap owns.
type TVShow struct {
	Title         string
	OriginalTitle string
	Year          *int
	Plot          string
	Genres        []string
	TMDbID        string
	PosterURL     string
	BackdropURL   string
	Rating        *float64
	Votes         *int
	Status        string
	Network       string
	Cast          []Person
}

type Episode struct {
	Title    string
	Plot     string
	Season   int
	Episode  int
	AirDate  string
	Runtime  *int
	TMDbID   string
	StillURL string
}

type tvShowXML struct {
	XMLName       xml.Name   `xml:"tvshow"`
	Title         string     `xml:"title"`
	OriginalTitle string     `xml:"originaltitle,omitempty"`
	Year          *int       `xml:"year,omitempty"`
	Plot          string     `xml:"plot,omitempty"`
	Rating        *float64   `xml:"rating,omitempty"`
	Votes         *int       `xml:"votes,omitempty"`
	Status        string     `xml:"status,omitempty"`
	Network       string     `xml:"studio,omitempty"`
	Genres        []string   `xml:"genre,omitempty"`
	Actors        []actorXML `xml:"actor,omitempty"`
	UniqueID      []idXML    `xml:"uniqueid"`
	Thumb         []thumbXML `xml:"thumb"`
}

type episodeXML struct {
	XMLName  xml.Name   `xml:"episodedetails"`
	Title    string     `xml:"title"`
	Plot     string     `xml:"plot,omitempty"`
	Season   int        `xml:"season"`
	Episode  int        `xml:"episode"`
	AirDate  string     `xml:"aired,omitempty"`
	Runtime  *int       `xml:"runtime,omitempty"`
	UniqueID []idXML    `xml:"uniqueid"`
	Thumb    []thumbXML `xml:"thumb"`
}

func ParseTVShow(input []byte) (TVShow, error) {
	var decoded tvShowXML
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = true
	if err := decoder.Decode(&decoded); err != nil {
		return TVShow{}, err
	}
	show := TVShow{
		Title:         strings.TrimSpace(decoded.Title),
		OriginalTitle: strings.TrimSpace(decoded.OriginalTitle),
		Year:          decoded.Year,
		Plot:          strings.TrimSpace(decoded.Plot),
		Genres:        cleaned(decoded.Genres),
		Rating:        decoded.Rating,
		Votes:         decoded.Votes,
		Status:        strings.TrimSpace(decoded.Status),
		Network:       strings.TrimSpace(decoded.Network),
	}
	for _, id := range decoded.UniqueID {
		if strings.EqualFold(id.Type, "tmdb") {
			show.TMDbID = strings.TrimSpace(id.Value)
			break
		}
	}
	for _, thumb := range decoded.Thumb {
		switch strings.ToLower(thumb.Aspect) {
		case "poster":
			show.PosterURL = strings.TrimSpace(thumb.Value)
		case "fanart":
			show.BackdropURL = strings.TrimSpace(thumb.Value)
		}
	}
	for _, actor := range decoded.Actors {
		if name := strings.TrimSpace(actor.Name); name != "" {
			show.Cast = append(show.Cast, Person{Name: name, Role: strings.TrimSpace(actor.Role), Thumb: strings.TrimSpace(actor.Thumb)})
		}
	}
	return show, nil
}

func WriteTVShow(show TVShow) ([]byte, error) {
	encoded := tvShowXML{
		Title:         strings.TrimSpace(show.Title),
		OriginalTitle: strings.TrimSpace(show.OriginalTitle),
		Year:          show.Year,
		Plot:          strings.TrimSpace(show.Plot),
		Genres:        cleaned(show.Genres),
		Rating:        show.Rating,
		Votes:         show.Votes,
		Status:        strings.TrimSpace(show.Status),
		Network:       strings.TrimSpace(show.Network),
	}
	if show.TMDbID != "" {
		encoded.UniqueID = []idXML{{Type: "tmdb", Default: "true", Value: show.TMDbID}}
	}
	if show.PosterURL != "" {
		encoded.Thumb = append(encoded.Thumb, thumbXML{Aspect: "poster", Value: show.PosterURL})
	}
	if show.BackdropURL != "" {
		encoded.Thumb = append(encoded.Thumb, thumbXML{Aspect: "fanart", Value: show.BackdropURL})
	}
	for _, person := range show.Cast {
		if name := strings.TrimSpace(person.Name); name != "" {
			encoded.Actors = append(encoded.Actors, actorXML{Name: name, Role: strings.TrimSpace(person.Role), Thumb: strings.TrimSpace(person.Thumb)})
		}
	}
	return writeXML(encoded)
}

func ParseEpisode(input []byte) (Episode, error) {
	var decoded episodeXML
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = true
	if err := decoder.Decode(&decoded); err != nil {
		return Episode{}, err
	}
	episode := Episode{
		Title:   strings.TrimSpace(decoded.Title),
		Plot:    strings.TrimSpace(decoded.Plot),
		Season:  decoded.Season,
		Episode: decoded.Episode,
		AirDate: strings.TrimSpace(decoded.AirDate),
		Runtime: decoded.Runtime,
	}
	for _, id := range decoded.UniqueID {
		if strings.EqualFold(id.Type, "tmdb") {
			episode.TMDbID = strings.TrimSpace(id.Value)
			break
		}
	}
	for _, thumb := range decoded.Thumb {
		if strings.EqualFold(thumb.Aspect, "thumb") {
			episode.StillURL = strings.TrimSpace(thumb.Value)
			break
		}
	}
	return episode, nil
}

func WriteEpisode(episode Episode) ([]byte, error) {
	encoded := episodeXML{
		Title:   strings.TrimSpace(episode.Title),
		Plot:    strings.TrimSpace(episode.Plot),
		Season:  episode.Season,
		Episode: episode.Episode,
		AirDate: strings.TrimSpace(episode.AirDate),
		Runtime: episode.Runtime,
	}
	if episode.TMDbID != "" {
		encoded.UniqueID = []idXML{{Type: "tmdb", Default: "true", Value: episode.TMDbID}}
	}
	if episode.StillURL != "" {
		encoded.Thumb = []thumbXML{{Aspect: "thumb", Value: episode.StillURL}}
	}
	return writeXML(encoded)
}

func WriteSeason(title string, seasonNumber int) ([]byte, error) {
	encoded := struct {
		XMLName xml.Name `xml:"season"`
		Title   string   `xml:"title"`
		Season  int      `xml:"season"`
	}{Title: strings.TrimSpace(title), Season: seasonNumber}
	return writeXML(encoded)
}
