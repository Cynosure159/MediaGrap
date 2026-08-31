package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	mu       sync.RWMutex
	client   *http.Client
	apiKey   string
	language string
	logger   *slog.Logger
}

func NewTMDb(logger *slog.Logger, client *http.Client, apiKey string) *TMDb {
	if logger == nil {
		logger = slog.Default()
	}
	return &TMDb{client: client, apiKey: strings.TrimSpace(apiKey), language: "en-US", logger: logger}
}
func (t *TMDb) Configure(client *http.Client, apiKey, language string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.client = client
	t.apiKey = strings.TrimSpace(apiKey)
	t.language = language
}
func (t *TMDb) HTTPClient() *http.Client {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.client
}
func (t *TMDb) Logger() *slog.Logger { return t.logger }
func (t *TMDb) configured() (string, string, *http.Client, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.apiKey == "" {
		return "", "", nil, errors.New("TMDb is not configured; add an API key in Settings")
	}
	return t.apiKey, t.language, t.client, nil
}
func (t *TMDb) SearchMovies(ctx context.Context, query string, year *int) ([]Candidate, error) {
	apiKey, language, _, err := t.configured()
	if err != nil {
		return nil, err
	}
	params := url.Values{"api_key": {apiKey}, "query": {query}, "language": {language}}
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
	apiKey, language, _, err := t.configured()
	if err != nil {
		return Details{}, err
	}
	if id == "" {
		return Details{}, errors.New("TMDb movie id is required")
	}
	var result tmdbMovie
	if err := t.get(ctx, "/movie/"+url.PathEscape(id), url.Values{"api_key": {apiKey}, "language": {language}}, &result); err != nil {
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
	startedAt := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.themoviedb.org/3"+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()
	response, err := client.Do(req)
	if err != nil {
		t.logger.Warn("TMDb request failed", "endpoint", path, "duration", time.Since(startedAt), "failure", requestFailureKind(err))
		return errors.New("TMDb request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.logger.Warn("TMDb request returned an error", "endpoint", path, "status", response.StatusCode, "duration", time.Since(startedAt))
		return fmt.Errorf("TMDb returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target); err != nil {
		t.logger.Warn("TMDb response could not be decoded", "endpoint", path, "status", response.StatusCode, "duration", time.Since(startedAt))
		return errors.New("TMDb returned invalid JSON")
	}
	t.logger.Info("TMDb request completed", "endpoint", path, "status", response.StatusCode, "duration", time.Since(startedAt))
	return nil
}

func requestFailureKind(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "transport"
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
	return &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}
