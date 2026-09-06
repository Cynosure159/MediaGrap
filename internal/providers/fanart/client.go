package fanart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const DefaultEndpoint = "https://webservice.fanart.tv/v3.2"

// Asset is the provider-neutral portion of a Fanart.tv movie artwork entry.
// The metadata package maps it to its own domain model before persistence.
type Asset struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Lang   string `json:"lang"`
	Season string `json:"season"`
	Likes  int    `json:"likes"`
	Added  string `json:"added"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Kind   string `json:"kind"`
}

// Fanart.tv has returned numeric fields as both JSON numbers and quoted
// strings across API versions, so normalize either representation here.
func (a *Asset) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID     string          `json:"id"`
		URL    string          `json:"url"`
		Lang   string          `json:"lang"`
		Season string          `json:"season"`
		Likes  json.RawMessage `json:"likes"`
		Added  string          `json:"added"`
		Width  json.RawMessage `json:"width"`
		Height json.RawMessage `json:"height"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	a.ID, a.URL, a.Lang, a.Season, a.Added = raw.ID, raw.URL, raw.Lang, raw.Season, raw.Added
	a.Likes, a.Width, a.Height = flexibleInt(raw.Likes), flexibleInt(raw.Width), flexibleInt(raw.Height)
	return nil
}

func flexibleInt(value json.RawMessage) int {
	if len(value) == 0 || string(value) == "null" {
		return 0
	}
	var number int
	if json.Unmarshal(value, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		number, _ = strconv.Atoi(strings.TrimSpace(text))
	}
	return number
}

type Client struct {
	logger    *slog.Logger
	endpoint  string
	client    *http.Client
	apiKey    string
	clientKey string
	language  string
	mu        sync.RWMutex
}

func NewClient(logger *slog.Logger, client *http.Client, apiKey string) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{logger: logger, endpoint: DefaultEndpoint, client: client, apiKey: strings.TrimSpace(apiKey), language: "en-US"}
}

func (c *Client) SetEndpoint(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endpoint = strings.TrimRight(endpoint, "/")
}

func (c *Client) Configure(client *http.Client, apiKey, language string) {
	c.ConfigureKeys(client, apiKey, "", language)
}

func (c *Client) ConfigureKeys(client *http.Client, apiKey, clientKey, language string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client != nil {
		c.client = client
	}
	c.apiKey = strings.TrimSpace(apiKey)
	c.clientKey = strings.TrimSpace(clientKey)
	if strings.TrimSpace(language) != "" {
		c.language = strings.TrimSpace(language)
	}
}

func (c *Client) configured() (string, string, string, string, *http.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.apiKey == "" {
		return "", "", "", "", nil, errors.New("Fanart.tv API key is not configured")
	}
	return c.apiKey, c.clientKey, c.language, c.endpoint, c.client, nil
}

func (c *Client) Ping(ctx context.Context) (int, error) {
	apiKey, clientKey, _, endpoint, client, err := c.configured()
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/movies/550", nil)
	if err != nil {
		return 0, err
	}
	setAuthorization(request, apiKey, clientKey)
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return 0, errors.New("Fanart.tv connection failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, fmt.Errorf("Fanart.tv returned HTTP %d", response.StatusCode)
	}
	return response.StatusCode, nil
}

// Movie returns the artwork groups supported by Kodi and Fanart.tv. The
// order follows MediaElch's useful behavior: preferred language first, then
// language-neutral artwork, English, likes, and finally provider order.
func (c *Client) Movie(ctx context.Context, tmdbID string) ([]Asset, error) {
	apiKey, clientKey, language, endpoint, client, err := c.configured()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tmdbID) == "" {
		return nil, errors.New("TMDb movie id is required for Fanart.tv")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/movies/"+url.PathEscape(strings.TrimSpace(tmdbID)), nil)
	if err != nil {
		return nil, errors.New("create Fanart.tv request")
	}
	setAuthorization(request, apiKey, clientKey)
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("Fanart.tv request failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return []Asset{}, nil
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("Fanart.tv rate limit exceeded")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Fanart.tv returned HTTP %d", response.StatusCode)
	}
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, errors.New("decode Fanart.tv response")
	}
	assets, err := mapArtwork(payload, movieArtworkGroups, language)
	if err != nil {
		return nil, err
	}
	c.logger.Info("Fanart.tv movie artwork mapped", "tmdb_id", tmdbID, "asset_count", len(assets))
	return assets, nil
}

// TV returns normalized show and season artwork. Fanart.tv identifies TV shows
// by TheTVDB ID, unlike its movie endpoint which uses a TMDb ID.
func (c *Client) TV(ctx context.Context, tvdbID string) ([]Asset, error) {
	apiKey, clientKey, language, endpoint, client, err := c.configured()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tvdbID) == "" {
		return nil, errors.New("TheTVDB TV id is required for Fanart.tv")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/tv/"+url.PathEscape(strings.TrimSpace(tvdbID)), nil)
	if err != nil {
		return nil, errors.New("create Fanart.tv TV request")
	}
	setAuthorization(request, apiKey, clientKey)
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("Fanart.tv TV request failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return []Asset{}, nil
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("Fanart.tv rate limit exceeded")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Fanart.tv returned HTTP %d", response.StatusCode)
	}
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, errors.New("decode Fanart.tv TV response")
	}
	assets, err := mapArtwork(payload, tvArtworkGroups, language)
	if err != nil {
		return nil, err
	}
	c.logger.Info("Fanart.tv TV artwork mapped", "tvdb_id", tvdbID, "asset_count", len(assets))
	return assets, nil
}

func setAuthorization(request *http.Request, apiKey, clientKey string) {
	if apiKey != "" {
		request.Header.Set("api-key", apiKey)
	}
	if clientKey != "" {
		request.Header.Set("client-key", clientKey)
	}
}

var movieArtworkGroups = [][2]string{
	{"movieposter", "poster"},
	{"moviebackground", "fanart"},
	{"hdmovielogo", "clearlogo"},
	{"hdmovieclearart", "clearart"},
	{"moviedisc", "discart"},
	{"moviebanner", "banner"},
	{"moviethumb", "landscape"},
}

var tvArtworkGroups = [][2]string{
	{"tvposter", "poster"},
	{"showbackground", "fanart"},
	{"clearlogo", "clearlogo"},
	{"hdtvlogo", "logo"},
	{"clearart", "clearart"},
	{"hdclearart", "clearart"},
	{"tvbanner", "banner"},
	{"tvthumb", "landscape"},
	{"seasonposter", "season_poster"},
	{"seasonbanner", "season_banner"},
	{"seasonthumb", "season_landscape"},
	{"characterart", "character"},
}

func mapArtwork(payload map[string]json.RawMessage, groups [][2]string, language string) ([]Asset, error) {
	assets := make([]Asset, 0)
	for _, group := range groups {
		providerGroup, kind := group[0], group[1]
		var entries []Asset
		if raw, ok := payload[providerGroup]; ok {
			if err := json.Unmarshal(raw, &entries); err != nil {
				return nil, fmt.Errorf("decode Fanart.tv %s artwork: %w", providerGroup, err)
			}
		}
		for index := range entries {
			entries[index].Kind = kind
			assets = append(assets, entries[index])
		}
	}
	sort.SliceStable(assets, func(i, j int) bool {
		if artworkKindRank(assets[i].Kind, groups) != artworkKindRank(assets[j].Kind, groups) {
			return artworkKindRank(assets[i].Kind, groups) < artworkKindRank(assets[j].Kind, groups)
		}
		if assets[i].Season != assets[j].Season {
			return assets[i].Season < assets[j].Season
		}
		left, right := languageScore(assets[i].Lang, language), languageScore(assets[j].Lang, language)
		if left != right {
			return left < right
		}
		if assets[i].Likes != assets[j].Likes {
			return assets[i].Likes > assets[j].Likes
		}
		return assets[i].ID < assets[j].ID
	})
	return assets, nil
}

func artworkKindRank(kind string, groups [][2]string) int {
	for index, group := range groups {
		if group[1] == kind {
			return index
		}
	}
	return len(groups)
}

func languageScore(value, preferred string) int {
	value = strings.TrimSpace(strings.ToLower(value))
	preferred = strings.ToLower(strings.TrimSpace(preferred))
	if value == preferred {
		return 0
	}
	if preferred != "" && strings.HasPrefix(preferred, value+"-") {
		return 1
	}
	if value == "00" || value == "" {
		return 2
	}
	if value == "en" || value == "en-us" {
		return 3
	}
	return 4
}

func (a Asset) String() string {
	return a.Kind + ":" + a.ID + " (" + strconv.Itoa(a.Width) + "x" + strconv.Itoa(a.Height) + ")"
}
