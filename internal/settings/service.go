package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var tmdbLanguagePattern = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)

type Defaults struct {
	TMDbAPIKey             string
	FanartTVAPIKey         string
	FanartTVPersonalAPIKey string
	TMDbLanguage           string
	FallbackLanguage       string
	OutboundProxy          string
	NoProxy                string
	MediaRoots             []string
}

type Snapshot struct {
	TMDbAPIKey             string
	FanartTVAPIKey         string
	FanartTVPersonalAPIKey string
	TMDbLanguage           string
	FallbackLanguage       string
	OutboundProxy          string
	NoProxy                string
	MediaRoots             []string
}

type View struct {
	TMDbAPIKeyConfigured             bool     `json:"tmdbApiKeyConfigured"`
	FanartTVAPIKeyConfigured         bool     `json:"fanartTvApiKeyConfigured"`
	FanartTVPersonalAPIKeyConfigured bool     `json:"fanartTvPersonalApiKeyConfigured"`
	OutboundProxyConfigured          bool     `json:"outboundProxyConfigured"`
	TMDbLanguage                     string   `json:"tmdbLanguage"`
	FallbackLanguage                 string   `json:"fallbackLanguage"`
	NoProxyConfigured                bool     `json:"noProxyConfigured"`
	Theme                            string   `json:"theme"`
	Locale                           string   `json:"locale"`
	MediaRoots                       []string `json:"mediaRoots"`
}

type Update struct {
	TMDbAPIKey                  string `json:"tmdbApiKey"`
	ClearTMDbAPIKey             bool   `json:"clearTmdbApiKey"`
	FanartTVAPIKey              string `json:"fanartTvApiKey"`
	ClearFanartTVAPIKey         bool   `json:"clearFanartTvApiKey"`
	FanartTVPersonalAPIKey      string `json:"fanartTvPersonalApiKey"`
	ClearFanartTVPersonalAPIKey bool   `json:"clearFanartTvPersonalApiKey"`
	TMDbLanguage                string `json:"tmdbLanguage"`
	FallbackLanguage            string `json:"fallbackLanguage"`
	OutboundProxy               string `json:"outboundProxy"`
	ClearOutboundProxy          bool   `json:"clearOutboundProxy"`
	NoProxy                     string `json:"noProxy"`
	ClearNoProxy                bool   `json:"clearNoProxy"`
	Theme                       string `json:"theme"`
	Locale                      string `json:"locale"`
}

type Service struct {
	db       *sql.DB
	defaults Defaults
}

func NewService(db *sql.DB, defaults Defaults) *Service {
	if defaults.TMDbLanguage == "" {
		defaults.TMDbLanguage = "en-US"
	}
	if defaults.FallbackLanguage == "" {
		defaults.FallbackLanguage = "en-US"
	}
	return &Service{db: db, defaults: defaults}
}

func (s *Service) Current(ctx context.Context) (Snapshot, error) {
	values := map[string]string{}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM application_settings WHERE key IN ('tmdb_api_key', 'fanart_tv_api_key', 'fanart_tv_personal_api_key', 'tmdb_language', 'fallback_language', 'outbound_proxy', 'no_proxy')`)
	if err != nil {
		return Snapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return Snapshot{}, err
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		TMDbAPIKey:             valueOrDefault(values, "tmdb_api_key", s.defaults.TMDbAPIKey),
		FanartTVAPIKey:         valueOrDefault(values, "fanart_tv_api_key", s.defaults.FanartTVAPIKey),
		FanartTVPersonalAPIKey: valueOrDefault(values, "fanart_tv_personal_api_key", s.defaults.FanartTVPersonalAPIKey),
		TMDbLanguage:           valueOrDefault(values, "tmdb_language", s.defaults.TMDbLanguage),
		FallbackLanguage:       valueOrDefault(values, "fallback_language", s.defaults.FallbackLanguage),
		OutboundProxy:          valueOrDefault(values, "outbound_proxy", s.defaults.OutboundProxy),
		NoProxy:                valueOrDefault(values, "no_proxy", s.defaults.NoProxy),
		MediaRoots:             append([]string(nil), s.defaults.MediaRoots...),
	}, nil
}

func (s *Service) View(ctx context.Context, userID ...int64) (View, error) {
	current, err := s.Current(ctx)
	if err != nil {
		return View{}, err
	}
	view := View{TMDbAPIKeyConfigured: current.TMDbAPIKey != "", FanartTVAPIKeyConfigured: current.FanartTVAPIKey != "", FanartTVPersonalAPIKeyConfigured: current.FanartTVPersonalAPIKey != "", OutboundProxyConfigured: current.OutboundProxy != "", NoProxyConfigured: current.NoProxy != "", TMDbLanguage: current.TMDbLanguage, FallbackLanguage: current.FallbackLanguage, Theme: "dark", Locale: "en", MediaRoots: current.MediaRoots}
	if len(userID) > 0 && userID[0] > 0 {
		_ = s.db.QueryRowContext(ctx, `SELECT theme, locale FROM user_preferences WHERE user_id=?`, userID[0]).Scan(&view.Theme, &view.Locale)
	}
	return view, nil
}

func (s *Service) Update(ctx context.Context, update Update, userID ...int64) (Snapshot, error) {
	current, err := s.Current(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if value := strings.TrimSpace(update.TMDbLanguage); value != "" {
		current.TMDbLanguage = value
	}
	if !tmdbLanguagePattern.MatchString(current.TMDbLanguage) {
		return Snapshot{}, errors.New("TMDb language must use a code such as zh-CN or en-US")
	}
	if value := strings.TrimSpace(update.FallbackLanguage); value != "" {
		current.FallbackLanguage = value
	}
	if !tmdbLanguagePattern.MatchString(current.FallbackLanguage) {
		return Snapshot{}, errors.New("fallback language must use a code such as zh-CN or en-US")
	}
	if update.ClearTMDbAPIKey {
		current.TMDbAPIKey = ""
	} else if value := strings.TrimSpace(update.TMDbAPIKey); value != "" {
		current.TMDbAPIKey = value
	}
	if update.ClearFanartTVAPIKey {
		current.FanartTVAPIKey = ""
	} else if value := strings.TrimSpace(update.FanartTVAPIKey); value != "" {
		current.FanartTVAPIKey = value
	}
	if update.ClearFanartTVPersonalAPIKey {
		current.FanartTVPersonalAPIKey = ""
	} else if value := strings.TrimSpace(update.FanartTVPersonalAPIKey); value != "" {
		current.FanartTVPersonalAPIKey = value
	}
	if update.ClearOutboundProxy {
		current.OutboundProxy = ""
	} else if value := strings.TrimSpace(update.OutboundProxy); value != "" {
		current.OutboundProxy = value
	}
	if err := validateProxy(current.OutboundProxy); err != nil {
		return Snapshot{}, err
	}
	if update.ClearNoProxy {
		current.NoProxy = ""
	} else if value := strings.TrimSpace(update.NoProxy); value != "" {
		current.NoProxy = value
	}
	if err := validateNoProxy(current.NoProxy); err != nil {
		return Snapshot{}, err
	}
	theme, locale := strings.TrimSpace(update.Theme), strings.TrimSpace(update.Locale)
	if theme != "" && theme != "dark" && theme != "light" && theme != "system" {
		return Snapshot{}, errors.New("theme must be dark, light, or system")
	}
	if locale != "" && locale != "en" && locale != "zh-CN" {
		return Snapshot{}, errors.New("interface language must be en or zh-CN")
	}
	transaction, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer transaction.Rollback()
	for key, value := range map[string]string{"tmdb_api_key": current.TMDbAPIKey, "fanart_tv_api_key": current.FanartTVAPIKey, "fanart_tv_personal_api_key": current.FanartTVPersonalAPIKey, "tmdb_language": current.TMDbLanguage, "fallback_language": current.FallbackLanguage, "outbound_proxy": current.OutboundProxy, "no_proxy": current.NoProxy} {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO application_settings(key, value, updated_at) VALUES(?, ?, CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`, key, value); err != nil {
			return Snapshot{}, fmt.Errorf("save setting: %w", err)
		}
	}
	if len(userID) > 0 && userID[0] > 0 && (theme != "" || locale != "") {
		currentTheme, currentLocale := "dark", "en"
		_ = transaction.QueryRowContext(ctx, `SELECT theme, locale FROM user_preferences WHERE user_id=?`, userID[0]).Scan(&currentTheme, &currentLocale)
		if theme != "" {
			currentTheme = theme
		}
		if locale != "" {
			currentLocale = locale
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO user_preferences(user_id,theme,locale,updated_at) VALUES(?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(user_id) DO UPDATE SET theme=excluded.theme,locale=excluded.locale,updated_at=excluded.updated_at`, userID[0], currentTheme, currentLocale); err != nil {
			return Snapshot{}, fmt.Errorf("save user preferences: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return Snapshot{}, err
	}
	return current, nil
}

func valueOrDefault(values map[string]string, key, fallback string) string {
	if value, ok := values[key]; ok {
		return value
	}
	return fallback
}

func validateProxy(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "socks5" && parsed.Scheme != "socks5h") {
		return errors.New("outbound proxy must be a valid HTTP, HTTPS, or SOCKS5 proxy URL")
	}
	return nil
}

func validateNoProxy(value string) error {
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.ContainsAny(entry, "/?#@") {
			return errors.New("NO_PROXY entries must be hostnames, domains, IP addresses, or host:port values")
		}
	}
	return nil
}
