package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
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

type MovieDetails struct {
	Candidate
	Runtime       *int     `json:"runtime"`
	Genres        []string `json:"genres"`
	BackdropURL   string   `json:"backdropUrl"`
	Rating        *float64 `json:"rating"`
	Votes         *int     `json:"votes"`
	ContentRating string   `json:"contentRating"`
	Directors     []string `json:"directors"`
	Writers       []string `json:"writers"`
	Studios       []string `json:"studios"`
	Cast          []Person `json:"cast"`
}

type Person struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	ProfileURL string `json:"profileUrl"`
}

type TVDetails struct {
	Candidate
	TVDBID      string   `json:"tvdbId"`
	IMDbID      string   `json:"imdbId"`
	Genres      []string `json:"genres"`
	BackdropURL string   `json:"backdropUrl"`
	Rating      *float64 `json:"rating"`
	Votes       *int     `json:"votes"`
	Status      string   `json:"status"`
	Network     string   `json:"network"`
	Cast        []Person `json:"cast"`
}

type TVEpisodeDetails struct {
	SeasonNumber   int      `json:"seasonNumber"`
	EpisodeNumber  int      `json:"episodeNumber"`
	Title          string   `json:"title"`
	Overview       string   `json:"overview"`
	AirDate        string   `json:"airDate"`
	RuntimeMinutes *int     `json:"runtimeMinutes"`
	StillURL       string   `json:"stillUrl"`
	Rating         *float64 `json:"rating"`
	Votes          *int     `json:"votes"`
}

type Client struct {
	logger           *slog.Logger
	endpoint         string
	mu               sync.RWMutex
	client           *http.Client
	apiKey           string
	language         string
	fallbackLanguage string
}

func NewClient(logger *slog.Logger, client *http.Client, apiKey string) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		logger:           logger,
		endpoint:         "https://api.themoviedb.org/3",
		client:           client,
		apiKey:           strings.TrimSpace(apiKey),
		language:         "en-US",
		fallbackLanguage: "en-US",
	}
}

func NewOutboundClient(proxy string, noProxy ...string) (*http.Client, error) {
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		client.Transport = &http.Transport{Proxy: proxyFromEnvironmentWithOverride(strings.Join(noProxy, ","))}
		return client, nil
	}
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound proxy: %w", err)
	}
	if proxyURL.Hostname() == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https" && proxyURL.Scheme != "socks5" && proxyURL.Scheme != "socks5h") {
		return nil, errors.New("invalid outbound proxy URL")
	}
	bypass := strings.Join(noProxy, ",")
	if proxyURL.Scheme == "socks5" || proxyURL.Scheme == "socks5h" {
		client.Transport = &http.Transport{DialContext: socks5DialContext(proxyURL, bypass)}
	} else {
		client.Transport = &http.Transport{Proxy: func(request *http.Request) (*url.URL, error) {
			if bypassProxy(request.URL.Hostname(), request.URL.Port(), bypass) {
				return nil, nil
			}
			return proxyURL, nil
		}}
	}
	return client, nil
}

func proxyFromEnvironmentWithOverride(noProxy string) func(*http.Request) (*url.URL, error) {
	return func(request *http.Request) (*url.URL, error) {
		if bypassProxy(request.URL.Hostname(), request.URL.Port(), noProxy) {
			return nil, nil
		}
		return http.ProxyFromEnvironment(request)
	}
}

func bypassProxy(host, port, value string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, raw := range strings.Split(value, ",") {
		entry := strings.ToLower(strings.TrimSpace(raw))
		if entry == "" {
			continue
		}
		if entry == "*" {
			return true
		}
		entryHost, entryPort := entry, ""
		if parsedHost, parsedPort, err := net.SplitHostPort(entry); err == nil {
			entryHost, entryPort = parsedHost, parsedPort
		}
		entryHost = strings.TrimPrefix(entryHost, ".")
		if entryPort != "" && entryPort != port {
			continue
		}
		if host == entryHost || strings.HasSuffix(host, "."+entryHost) {
			return true
		}
	}
	return false
}

func socks5DialContext(proxyURL *url.URL, noProxy string) func(context.Context, string, string) (net.Conn, error) {
	proxyAddress := proxyURL.Host
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if bypassProxy(host, port, noProxy) {
			return (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, address)
		}
		conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", proxyAddress)
		if err != nil {
			return nil, err
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		if err = socks5Handshake(conn, proxyURL, host, port); err != nil {
			_ = conn.Close()
			return nil, err
		}
		_ = conn.SetDeadline(time.Time{})
		return conn, nil
	}
}

func socks5Handshake(conn net.Conn, proxyURL *url.URL, host, port string) error {
	if len(host) > 255 {
		return errors.New("SOCKS5 destination hostname is too long")
	}
	methods := []byte{0x00}
	if proxyURL.User != nil {
		methods = append(methods, 0x02)
	}
	if _, err := conn.Write(append([]byte{0x05, byte(len(methods))}, methods...)); err != nil {
		return err
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil || reply[0] != 0x05 || reply[1] == 0xff {
		return errors.New("SOCKS5 proxy authentication negotiation failed")
	}
	if reply[1] == 0x02 {
		password, _ := proxyURL.User.Password()
		username := proxyURL.User.Username()
		if len(username) > 255 || len(password) > 255 {
			return errors.New("SOCKS5 proxy credentials are too long")
		}
		request := append([]byte{0x01, byte(len(username))}, []byte(username)...)
		request = append(request, byte(len(password)))
		request = append(request, []byte(password)...)
		if _, err := conn.Write(request); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, reply); err != nil || reply[1] != 0x00 {
			return errors.New("SOCKS5 proxy authentication failed")
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("invalid SOCKS5 destination port")
	}
	request := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	request = append(request, []byte(host)...)
	request = append(request, byte(portNumber>>8), byte(portNumber))
	if _, err := conn.Write(request); err != nil {
		return err
	}
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil || header[0] != 0x05 || header[1] != 0x00 {
		return errors.New("SOCKS5 proxy connection failed")
	}
	length := 0
	switch header[3] {
	case 0x01:
		length = 4
	case 0x04:
		length = 16
	case 0x03:
		size := make([]byte, 1)
		if _, err := io.ReadFull(conn, size); err != nil {
			return err
		}
		length = int(size[0])
	default:
		return errors.New("invalid SOCKS5 proxy response")
	}
	_, err = io.CopyN(io.Discard, conn, int64(length+2))
	return err
}

func (c *Client) Ping(ctx context.Context) (int, error) {
	apiKey, _, endpoint, err := c.configured()
	if err != nil {
		return 0, err
	}
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/configuration?api_key="+url.QueryEscape(apiKey), nil)
	if err != nil {
		return 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, errors.New("TMDb connection failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fmt.Errorf("TMDb returned HTTP %d", response.StatusCode)
	}
	return response.StatusCode, nil
}

func (c *Client) SetEndpoint(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endpoint = strings.TrimRight(endpoint, "/")
}

func (c *Client) Configure(client *http.Client, apiKey, language string) {
	c.ConfigureAdvanced(client, apiKey, language, "en-US")
}

func (c *Client) ConfigureAdvanced(client *http.Client, apiKey, language, fallbackLanguage string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client != nil {
		c.client = client
	}
	c.apiKey = strings.TrimSpace(apiKey)
	if strings.TrimSpace(language) != "" {
		c.language = strings.TrimSpace(language)
	}
	if strings.TrimSpace(fallbackLanguage) != "" {
		c.fallbackLanguage = strings.TrimSpace(fallbackLanguage)
	}
}

func (c *Client) fallback() string { c.mu.RLock(); defer c.mu.RUnlock(); return c.fallbackLanguage }

func (c *Client) HTTPClient() *http.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

func (c *Client) Logger() *slog.Logger {
	return c.logger
}

func (c *Client) configured() (apiKey, language, endpoint string, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.apiKey == "" {
		return "", "", "", errors.New("TMDb API key is not configured")
	}
	return c.apiKey, c.language, c.endpoint, nil
}

func (c *Client) SearchMovies(ctx context.Context, query string, year *int) ([]Candidate, error) {
	apiKey, language, _, err := c.configured()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is required")
	}
	params := url.Values{"api_key": {apiKey}, "query": {query}, "language": {language}}
	if year != nil && *year > 0 {
		params.Set("year", strconv.Itoa(*year))
	}
	var result struct {
		Results []tmdbMovie `json:"results"`
	}
	if err := c.get(ctx, "/search/movie", params, &result); err != nil {
		return nil, err
	}
	candidates := make([]Candidate, 0, len(result.Results))
	for _, item := range result.Results {
		candidates = append(candidates, item.candidate())
	}
	return candidates, nil
}

func (c *Client) Movie(ctx context.Context, id string) (MovieDetails, error) {
	apiKey, language, _, err := c.configured()
	if err != nil {
		return MovieDetails{}, err
	}
	if id == "" {
		return MovieDetails{}, errors.New("TMDb movie id is required")
	}
	var result tmdbMovie
	params := url.Values{"api_key": {apiKey}, "language": {language}, "append_to_response": {"credits,release_dates"}}
	if err := c.get(ctx, "/movie/"+url.PathEscape(id), params, &result); err != nil {
		return MovieDetails{}, err
	}
	if fallback := c.fallback(); strings.TrimSpace(result.Overview) == "" && fallback != "" && fallback != language {
		var translated tmdbMovie
		fallbackParams := url.Values{"api_key": {apiKey}, "language": {fallback}, "append_to_response": {"credits,release_dates"}}
		if c.get(ctx, "/movie/"+url.PathEscape(id), fallbackParams, &translated) == nil {
			fillMovieTranslation(&result, translated)
		}
	}
	genres := make([]string, 0, len(result.Genres))
	for _, genre := range result.Genres {
		genres = append(genres, genre.Name)
	}
	studios := make([]string, 0, len(result.ProductionCompanies))
	for _, company := range result.ProductionCompanies {
		if name := strings.TrimSpace(company.Name); name != "" {
			studios = append(studios, name)
		}
	}
	directors, writers := splitCrew(result.Credits.Crew)
	cast := make([]Person, 0, min(len(result.Credits.Cast), 20))
	for index, person := range result.Credits.Cast {
		if index == 20 {
			break
		}
		if name := strings.TrimSpace(person.Name); name != "" {
			cast = append(cast, Person{Name: name, Role: strings.TrimSpace(person.Character), ProfileURL: profileURL(person.ProfilePath)})
		}
	}
	contentRating := movieCertification(result.ReleaseDates.Results, language)
	details := MovieDetails{Candidate: result.candidate(), Runtime: optionalInt(result.Runtime), Genres: genres, BackdropURL: imageURL(result.BackdropPath), Rating: optionalFloat(result.VoteAverage), Votes: optionalInt(result.VoteCount), ContentRating: contentRating, Directors: directors, Writers: writers, Studios: studios, Cast: cast}
	c.logger.Info("TMDb movie details mapped", "movie_id", id, "cast_count", len(cast), "director_count", len(directors), "has_rating", details.Rating != nil)
	return details, nil
}

func (c *Client) SearchTV(ctx context.Context, query string, year *int) ([]Candidate, error) {
	apiKey, language, _, err := c.configured()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is required")
	}
	params := url.Values{"api_key": {apiKey}, "query": {query}, "language": {language}}
	if year != nil && *year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(*year))
	}
	var result struct {
		Results []tmdbTV `json:"results"`
	}
	if err := c.get(ctx, "/search/tv", params, &result); err != nil {
		return nil, err
	}
	candidates := make([]Candidate, 0, len(result.Results))
	for _, item := range result.Results {
		candidates = append(candidates, item.candidate())
	}
	return candidates, nil
}

func (c *Client) TV(ctx context.Context, id string) (TVDetails, error) {
	apiKey, language, _, err := c.configured()
	if err != nil {
		return TVDetails{}, err
	}
	if id == "" {
		return TVDetails{}, errors.New("TMDb TV id is required")
	}
	var result tmdbTV
	params := url.Values{"api_key": {apiKey}, "language": {language}, "append_to_response": {"credits"}}
	if err := c.get(ctx, "/tv/"+url.PathEscape(id), params, &result); err != nil {
		return TVDetails{}, err
	}
	var externalIDs struct {
		TVDBID int    `json:"tvdb_id"`
		IMDbID string `json:"imdb_id"`
	}
	if err := c.get(ctx, "/tv/"+url.PathEscape(id)+"/external_ids", url.Values{"api_key": {apiKey}}, &externalIDs); err != nil {
		c.logger.Warn("TMDb TV external IDs lookup failed", "tv_id", id, "error", err)
	}
	if fallback := c.fallback(); strings.TrimSpace(result.Overview) == "" && fallback != "" && fallback != language {
		var translated tmdbTV
		fallbackParams := url.Values{"api_key": {apiKey}, "language": {fallback}, "append_to_response": {"credits"}}
		if c.get(ctx, "/tv/"+url.PathEscape(id), fallbackParams, &translated) == nil {
			fillTVTranslation(&result, translated)
		}
	}
	genres := make([]string, 0, len(result.Genres))
	for _, genre := range result.Genres {
		genres = append(genres, genre.Name)
	}
	cast := make([]Person, 0, min(len(result.Credits.Cast), 20))
	for index, person := range result.Credits.Cast {
		if index == 20 {
			break
		}
		if name := strings.TrimSpace(person.Name); name != "" {
			cast = append(cast, Person{Name: name, Role: strings.TrimSpace(person.Character), ProfileURL: profileURL(person.ProfilePath)})
		}
	}
	network := ""
	if len(result.Networks) > 0 {
		network = strings.TrimSpace(result.Networks[0].Name)
	}
	details := TVDetails{Candidate: result.candidate(), TVDBID: strconv.Itoa(externalIDs.TVDBID), IMDbID: strings.TrimSpace(externalIDs.IMDbID), Genres: genres, BackdropURL: imageURL(result.BackdropPath), Rating: optionalFloat(result.VoteAverage), Votes: optionalInt(result.VoteCount), Status: strings.TrimSpace(result.Status), Network: network, Cast: cast}
	if externalIDs.TVDBID == 0 {
		details.TVDBID = ""
	}
	c.logger.Info("TMDb TV details mapped", "tv_id", id, "tvdb_id", details.TVDBID, "season_count", len(result.Seasons), "cast_count", len(cast), "has_rating", details.Rating != nil)
	return details, nil
}

func (c *Client) TVSeason(ctx context.Context, id string, seasonNumber int) ([]TVEpisodeDetails, error) {
	apiKey, language, _, err := c.configured()
	if err != nil {
		return nil, err
	}
	if seasonNumber < 0 {
		return nil, errors.New("TMDb season number is invalid")
	}
	var result struct {
		Episodes []tmdbTVEpisode `json:"episodes"`
	}
	params := url.Values{"api_key": {apiKey}, "language": {language}}
	if err := c.get(ctx, "/tv/"+url.PathEscape(id)+"/season/"+strconv.Itoa(seasonNumber), params, &result); err != nil {
		return nil, err
	}
	episodes := make([]TVEpisodeDetails, 0, len(result.Episodes))
	for _, episode := range result.Episodes {
		episodes = append(episodes, TVEpisodeDetails{SeasonNumber: seasonNumber, EpisodeNumber: episode.EpisodeNumber, Title: strings.TrimSpace(episode.Name), Overview: strings.TrimSpace(episode.Overview), AirDate: strings.TrimSpace(episode.AirDate), RuntimeMinutes: optionalInt(episode.Runtime()), StillURL: imageURL(episode.StillPath)})
	}
	return episodes, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, target any) error {
	c.mu.RLock()
	httpClient := c.client
	endpoint := c.endpoint
	c.mu.RUnlock()

	requestURL := endpoint + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create TMDb request: %w", err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute TMDb request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return errors.New("TMDb resource not found")
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("TMDb returned status %d: %s", res.StatusCode, string(body))
	}
	return json.NewDecoder(res.Body).Decode(target)
}

type tmdbMovie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	ProductionCompanies []struct {
		Name string `json:"name"`
	} `json:"production_companies"`
	Credits struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
		} `json:"cast"`
		Crew []struct {
			Name       string `json:"name"`
			Job        string `json:"job"`
			Department string `json:"department"`
		} `json:"crew"`
	} `json:"credits"`
	ReleaseDates struct {
		Results []struct {
			Iso31661     string `json:"iso_3166_1"`
			ReleaseDates []struct {
				Certification string `json:"certification"`
			} `json:"release_dates"`
		} `json:"results"`
	} `json:"release_dates"`
}

func fillMovieTranslation(target *tmdbMovie, fallback tmdbMovie) {
	if strings.TrimSpace(target.Title) == "" {
		target.Title = fallback.Title
	}
	if strings.TrimSpace(target.OriginalTitle) == "" {
		target.OriginalTitle = fallback.OriginalTitle
	}
	if strings.TrimSpace(target.Overview) == "" {
		target.Overview = fallback.Overview
	}
}

func (m tmdbMovie) candidate() Candidate {
	var year *int
	if len(m.ReleaseDate) >= 4 {
		if parsed, err := strconv.Atoi(m.ReleaseDate[:4]); err == nil && parsed > 0 {
			year = &parsed
		}
	}
	return Candidate{
		ID:            strconv.Itoa(m.ID),
		Title:         strings.TrimSpace(m.Title),
		OriginalTitle: strings.TrimSpace(m.OriginalTitle),
		Year:          year,
		Overview:      strings.TrimSpace(m.Overview),
		PosterURL:     imageURL(m.PosterPath),
	}
}

type tmdbTV struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	FirstAirDate string  `json:"first_air_date"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
	Status       string  `json:"status"`
	Genres       []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Networks []struct {
		Name string `json:"name"`
	} `json:"networks"`
	Seasons []struct {
		SeasonNumber int `json:"season_number"`
	} `json:"seasons"`
	Credits struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
		} `json:"cast"`
	} `json:"credits"`
}

func fillTVTranslation(target *tmdbTV, fallback tmdbTV) {
	if strings.TrimSpace(target.Name) == "" {
		target.Name = fallback.Name
	}
	if strings.TrimSpace(target.OriginalName) == "" {
		target.OriginalName = fallback.OriginalName
	}
	if strings.TrimSpace(target.Overview) == "" {
		target.Overview = fallback.Overview
	}
}

func (v tmdbTV) candidate() Candidate {
	var year *int
	if len(v.FirstAirDate) >= 4 {
		if parsed, err := strconv.Atoi(v.FirstAirDate[:4]); err == nil && parsed > 0 {
			year = &parsed
		}
	}
	return Candidate{
		ID:            strconv.Itoa(v.ID),
		Title:         strings.TrimSpace(v.Name),
		OriginalTitle: strings.TrimSpace(v.OriginalName),
		Year:          year,
		Overview:      strings.TrimSpace(v.Overview),
		PosterURL:     imageURL(v.PosterPath),
	}
}

type tmdbTVEpisode struct {
	EpisodeNumber  int    `json:"episode_number"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	AirDate        string `json:"air_date"`
	StillPath      string `json:"still_path"`
	RuntimeMinutes int    `json:"runtime"`
}

func (e tmdbTVEpisode) Runtime() int {
	return e.RuntimeMinutes
}

func splitCrew(crew []struct {
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}) (directors, writers []string) {
	dMap := make(map[string]struct{})
	wMap := make(map[string]struct{})
	for _, member := range crew {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		if member.Job == "Director" {
			dMap[name] = struct{}{}
		}
		if member.Job == "Writer" || member.Job == "Screenplay" || member.Department == "Writing" {
			wMap[name] = struct{}{}
		}
	}
	for d := range dMap {
		directors = append(directors, d)
	}
	for w := range wMap {
		writers = append(writers, w)
	}
	return directors, writers
}

func movieCertification(results []struct {
	Iso31661     string `json:"iso_3166_1"`
	ReleaseDates []struct {
		Certification string `json:"certification"`
	} `json:"release_dates"`
}, language string) string {
	country := "US"
	if parts := strings.Split(language, "-"); len(parts) == 2 {
		country = strings.ToUpper(parts[1])
	}
	for _, entry := range results {
		if entry.Iso31661 == country {
			for _, rd := range entry.ReleaseDates {
				if cert := strings.TrimSpace(rd.Certification); cert != "" {
					return cert
				}
			}
		}
	}
	for _, entry := range results {
		if entry.Iso31661 == "US" {
			for _, rd := range entry.ReleaseDates {
				if cert := strings.TrimSpace(rd.Certification); cert != "" {
					return cert
				}
			}
		}
	}
	return ""
}

func imageURL(path string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/original" + path
}

func profileURL(path string) string {
	if path == "" {
		return ""
	}
	return "https://image.tmdb.org/t/p/w185" + path
}

func optionalInt(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

func optionalFloat(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}
