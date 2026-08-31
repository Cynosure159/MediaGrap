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
	TMDbAPIKey    string
	TMDbLanguage  string
	OutboundProxy string
	MediaRoots    []string
}

type Snapshot struct {
	TMDbAPIKey    string
	TMDbLanguage  string
	OutboundProxy string
	MediaRoots    []string
}

type View struct {
	TMDbAPIKeyConfigured    bool     `json:"tmdbApiKeyConfigured"`
	OutboundProxyConfigured bool     `json:"outboundProxyConfigured"`
	TMDbLanguage            string   `json:"tmdbLanguage"`
	MediaRoots              []string `json:"mediaRoots"`
}

type Update struct {
	TMDbAPIKey         string `json:"tmdbApiKey"`
	ClearTMDbAPIKey    bool   `json:"clearTmdbApiKey"`
	TMDbLanguage       string `json:"tmdbLanguage"`
	OutboundProxy      string `json:"outboundProxy"`
	ClearOutboundProxy bool   `json:"clearOutboundProxy"`
}

type Service struct {
	db       *sql.DB
	defaults Defaults
}

func NewService(db *sql.DB, defaults Defaults) *Service {
	if defaults.TMDbLanguage == "" {
		defaults.TMDbLanguage = "en-US"
	}
	return &Service{db: db, defaults: defaults}
}

func (s *Service) Current(ctx context.Context) (Snapshot, error) {
	values := map[string]string{}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM application_settings WHERE key IN ('tmdb_api_key', 'tmdb_language', 'outbound_proxy')`)
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
		TMDbAPIKey:    valueOrDefault(values, "tmdb_api_key", s.defaults.TMDbAPIKey),
		TMDbLanguage:  valueOrDefault(values, "tmdb_language", s.defaults.TMDbLanguage),
		OutboundProxy: valueOrDefault(values, "outbound_proxy", s.defaults.OutboundProxy),
		MediaRoots:    append([]string(nil), s.defaults.MediaRoots...),
	}, nil
}

func (s *Service) View(ctx context.Context) (View, error) {
	current, err := s.Current(ctx)
	if err != nil {
		return View{}, err
	}
	return View{TMDbAPIKeyConfigured: current.TMDbAPIKey != "", OutboundProxyConfigured: current.OutboundProxy != "", TMDbLanguage: current.TMDbLanguage, MediaRoots: current.MediaRoots}, nil
}

func (s *Service) Update(ctx context.Context, update Update) (Snapshot, error) {
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
	if update.ClearTMDbAPIKey {
		current.TMDbAPIKey = ""
	} else if value := strings.TrimSpace(update.TMDbAPIKey); value != "" {
		current.TMDbAPIKey = value
	}
	if update.ClearOutboundProxy {
		current.OutboundProxy = ""
	} else if value := strings.TrimSpace(update.OutboundProxy); value != "" {
		current.OutboundProxy = value
	}
	if err := validateProxy(current.OutboundProxy); err != nil {
		return Snapshot{}, err
	}
	transaction, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer transaction.Rollback()
	for key, value := range map[string]string{"tmdb_api_key": current.TMDbAPIKey, "tmdb_language": current.TMDbLanguage, "outbound_proxy": current.OutboundProxy} {
		if _, err := transaction.ExecContext(ctx, `INSERT INTO application_settings(key, value, updated_at) VALUES(?, ?, CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`, key, value); err != nil {
			return Snapshot{}, fmt.Errorf("save setting: %w", err)
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
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("outbound proxy must be a valid HTTP or HTTPS proxy URL")
	}
	return nil
}
