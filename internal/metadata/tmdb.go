package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Candidate struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"originalTitle"`
	Year          *int   `json:"year"`
	Overview      string `json:"overview"`
	PosterURL     string `json:"posterUrl"`
}
type Details struct {
	Candidate
	Runtime     *int     `json:"runtime"`
	Genres      []string `json:"genres"`
	BackdropURL string   `json:"backdropUrl"`
}

type Provider interface {
	SearchMovies(context.Context, string, *int) ([]Candidate, error)
	Movie(context.Context, string) (Details, error)
}
type TMDb struct {
	client *http.Client
	apiKey string
}

func NewTMDb(client *http.Client, apiKey string) *TMDb {
	return &TMDb{client: client, apiKey: strings.TrimSpace(apiKey)}
}
func (t *TMDb) configured() error {
	if t.apiKey == "" {
		return errors.New("TMDb is not configured; set MEDIAGRAP_TMDB_API_KEY")
	}
	return nil
}
func (t *TMDb) SearchMovies(ctx context.Context, query string, year *int) ([]Candidate, error) {
	if err := t.configured(); err != nil {
		return nil, err
	}
	params := url.Values{"api_key": {t.apiKey}, "query": {query}, "language": {"en-US"}}
	if year != nil {
		params.Set("year", strconv.Itoa(*year))
	}
	var result struct {
		Results []tmdbMovie `json:"results"`
	}
	if err := t.get(ctx, "/search/movie", params, &result); err != nil {
		return nil, err
	}
	candidates := make([]Candidate, 0, len(result.Results))
	for _, item := range result.Results {
		candidates = append(candidates, item.candidate())
	}
	return candidates, nil
}
func (t *TMDb) Movie(ctx context.Context, id string) (Details, error) {
	if err := t.configured(); err != nil {
		return Details{}, err
	}
	if id == "" {
		return Details{}, errors.New("TMDb movie id is required")
	}
	var result tmdbMovie
	if err := t.get(ctx, "/movie/"+url.PathEscape(id), url.Values{"api_key": {t.apiKey}, "language": {"en-US"}}, &result); err != nil {
		return Details{}, err
	}
	candidate := result.candidate()
	genres := make([]string, 0, len(result.Genres))
	for _, genre := range result.Genres {
		genres = append(genres, genre.Name)
	}
	return Details{Candidate: candidate, Runtime: optionalInt(result.Runtime), Genres: genres, BackdropURL: imageURL(result.BackdropPath)}, nil
}
func (t *TMDb) get(ctx context.Context, path string, params url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3"+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	response, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("TMDb request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("TMDb returned HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target)
}

type tmdbMovie struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	BackdropPath  string `json:"backdrop_path"`
	Runtime       int    `json:"runtime"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

func (m tmdbMovie) candidate() Candidate {
	return Candidate{ID: strconv.Itoa(m.ID), Title: m.Title, OriginalTitle: m.OriginalTitle, Year: releaseYear(m.ReleaseDate), Overview: m.Overview, PosterURL: imageURL(m.PosterPath)}
}
func releaseYear(value string) *int {
	if len(value) < 4 {
		return nil
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return nil
	}
	return &year
}
func imageURL(path string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/w500" + path
}
func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func NewOutboundClient(proxyValue string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(proxyValue) != "" {
		proxyURL, err := url.Parse(proxyValue)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" {
			return nil, errors.New("only HTTP and HTTPS proxy URLs are supported in this build")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Transport: transport, Timeout: 20 * time.Second}, nil
}
