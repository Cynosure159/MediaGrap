package app

import "testing"

func TestLoadConfigUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("MEDIAGRAP_CONFIG_DIR", "/tmp/mediagrap-config")
	t.Setenv("MEDIAGRAP_CACHE_DIR", "/tmp/mediagrap-cache")
	t.Setenv("MEDIAGRAP_LISTEN", "127.0.0.1:8080")

	config, err := LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.DatabasePath() != "/tmp/mediagrap-config/mediagrap.db" {
		t.Fatalf("unexpected database path %q", config.DatabasePath())
	}
}

func TestLoadConfigCanDisableFFprobe(t *testing.T) {
	t.Setenv("MEDIAGRAP_CONFIG_DIR", "/tmp/mediagrap-config")
	t.Setenv("MEDIAGRAP_CACHE_DIR", "/tmp/mediagrap-cache")
	t.Setenv("MEDIAGRAP_FFPROBE_PATH", "off")
	config, err := LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if config.FFprobePath != "" {
		t.Fatalf("expected ffprobe to be disabled, got %q", config.FFprobePath)
	}
}

func TestSecureSessionCookieConfiguration(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		t.Setenv("MEDIAGRAP_SECURE_SESSION_COOKIE", value)
		cfg, err := LoadConfig(nil)
		if err != nil || cfg.SecureSessionCookie != (value == "true") {
			t.Fatalf("cookie policy %s: %v", value, err)
		}
	}
	t.Setenv("MEDIAGRAP_SECURE_SESSION_COOKIE", "invalid")
	if _, err := LoadConfig(nil); err == nil {
		t.Fatal("invalid bool accepted")
	}
	t.Setenv("MEDIAGRAP_SECURE_SESSION_COOKIE", "false")
	cfg, err := LoadConfig([]string{"--secure-session-cookie=true"})
	if err != nil || !cfg.SecureSessionCookie {
		t.Fatal("flag must override environment")
	}
}
