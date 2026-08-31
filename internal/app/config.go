package app

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigDir = "/config"
	defaultCacheDir  = "/cache"
	defaultListen    = ":8080"
)

type Config struct {
	ConfigDir string
	CacheDir  string
	Listen    string
	LogFormat string
	LogLevel  string
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

	return config, nil
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
