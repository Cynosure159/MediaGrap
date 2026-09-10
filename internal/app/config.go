package app

import (
	"errors"
	"flag"
	"fmt"
	"github.com/mediagrap/mediagrap/internal/webhooks"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultConfigDir = "/config"
	defaultCacheDir  = "/cache"
	defaultListen    = ":8080"
)

type Config struct {
	IntegrationBacklogLimit int
	ConfigDir               string
	CacheDir                string
	Listen                  string
	LogFormat               string
	LogLevel                string
	MediaRoots              []string
	TMDbAPIKey              string
	FanartTVAPIKey          string
	FanartTVPersonalAPIKey  string
	TMDbLanguage            string
	FallbackLanguage        string
	OutboundProxy           string
	NoProxy                 string
	FFprobePath             string
	Webhooks                webhooks.Config
	MCPOrigins              []string
}

func LoadConfig(args []string) (Config, error) {
	config := Config{}
	flags := flag.NewFlagSet("mediagrap", flag.ContinueOnError)
	flags.StringVar(&config.ConfigDir, "config-dir", envOrDefault("MEDIAGRAP_CONFIG_DIR", defaultConfigDir), "persistent configuration directory")
	flags.StringVar(&config.CacheDir, "cache-dir", envOrDefault("MEDIAGRAP_CACHE_DIR", defaultCacheDir), "cache directory")
	flags.StringVar(&config.Listen, "listen", envOrDefault("MEDIAGRAP_LISTEN", defaultListen), "HTTP listen address")
	flags.StringVar(&config.LogFormat, "log-format", envOrDefault("MEDIAGRAP_LOG_FORMAT", "json"), "log format: json or text")
	flags.StringVar(&config.LogLevel, "log-level", envOrDefault("MEDIAGRAP_LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	config.ConfigDir = filepath.Clean(config.ConfigDir)
	config.CacheDir = filepath.Clean(config.CacheDir)
	if config.ConfigDir == "." || config.CacheDir == "." {
		return Config{}, errors.New("config and cache directories must not be the current directory")
	}
	if _, _, err := net.SplitHostPort(config.Listen); err != nil {
		return Config{}, fmt.Errorf("invalid listen address %q: %w", config.Listen, err)
	}
	if config.LogFormat != "json" && config.LogFormat != "text" {
		return Config{}, fmt.Errorf("invalid log format %q", config.LogFormat)
	}
	config.MediaRoots = cleanMediaRoots(envOrDefault("MEDIAGRAP_MEDIA_ROOTS", "/media"))
	config.TMDbAPIKey = strings.TrimSpace(os.Getenv("MEDIAGRAP_TMDB_API_KEY"))
	config.FanartTVAPIKey = strings.TrimSpace(os.Getenv("MEDIAGRAP_FANARTTV_API_KEY"))
	config.FanartTVPersonalAPIKey = strings.TrimSpace(os.Getenv("MEDIAGRAP_FANARTTV_PERSONAL_API_KEY"))
	config.TMDbLanguage = envOrDefault("MEDIAGRAP_TMDB_LANGUAGE", "en-US")
	config.FallbackLanguage = envOrDefault("MEDIAGRAP_FALLBACK_LANGUAGE", "en-US")
	config.OutboundProxy = strings.TrimSpace(os.Getenv("MEDIAGRAP_OUTBOUND_PROXY"))
	config.NoProxy = envOrDefault("MEDIAGRAP_NO_PROXY", os.Getenv("NO_PROXY"))
	config.IntegrationBacklogLimit = 100000
	if value := os.Getenv("MEDIAGRAP_INTEGRATION_BACKLOG_LIMIT"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 100 {
			return Config{}, errors.New("invalid integration backlog limit")
		}
		config.IntegrationBacklogLimit = limit
	}
	config.Webhooks.KeyFile = strings.TrimSpace(os.Getenv("MEDIAGRAP_WEBHOOK_KEY_FILE"))
	config.Webhooks.Paused = os.Getenv("MEDIAGRAP_WEBHOOK_PAUSED") == "true"
	config.Webhooks.Policy.AllowHTTP = os.Getenv("MEDIAGRAP_WEBHOOK_ALLOW_HTTP") == "true"
	for _, target := range strings.Split(os.Getenv("MEDIAGRAP_WEBHOOK_ALLOWED_TARGETS"), ",") {
		if target = strings.TrimSpace(target); target != "" {
			if _, _, err := net.SplitHostPort(target); err != nil {
				return Config{}, errors.New("invalid webhook allowed target")
			}
			config.Webhooks.Policy.AllowedTargets = append(config.Webhooks.Policy.AllowedTargets, target)
		}
	}
	for _, raw := range strings.Split(os.Getenv("MEDIAGRAP_WEBHOOK_ALLOWED_CIDRS"), ",") {
		if raw = strings.TrimSpace(raw); raw != "" {
			prefix, err := netip.ParsePrefix(raw)
			if err != nil {
				return Config{}, errors.New("invalid webhook allowed CIDR")
			}
			config.Webhooks.Policy.AllowedCIDRs = append(config.Webhooks.Policy.AllowedCIDRs, prefix)
		}
	}
	for _, origin := range strings.Split(os.Getenv("MEDIAGRAP_MCP_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			config.MCPOrigins = append(config.MCPOrigins, origin)
		}
	}
	config.FFprobePath = envOrDefault("MEDIAGRAP_FFPROBE_PATH", "ffprobe")
	if strings.EqualFold(config.FFprobePath, "off") || strings.EqualFold(config.FFprobePath, "disabled") {
		config.FFprobePath = ""
	}
	if len(config.MediaRoots) == 0 {
		return Config{}, errors.New("at least one media root is required")
	}

	return config, nil
}

func cleanMediaRoots(value string) []string {
	seen := make(map[string]struct{})
	roots := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		root := filepath.Clean(strings.TrimSpace(part))
		if root == "." || root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		roots = append(roots, root)
	}
	return roots
}

func (c Config) DatabasePath() string {
	return filepath.Join(c.ConfigDir, "mediagrap.db")
}

func (c Config) EnsureDirectories() error {
	for _, directory := range []string{c.ConfigDir, c.CacheDir} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return fmt.Errorf("create %s: %w", directory, err)
		}
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
