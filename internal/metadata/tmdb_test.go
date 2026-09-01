package metadata

import (
	"log/slog"
	"net/http"
	"testing"
)

func TestTMDbProviderInterface(t *testing.T) {
	provider := NewTMDb(slog.Default(), http.DefaultClient, "test-api-key")
	var _ Provider = provider
	var _ TVProvider = provider

	if provider.HTTPClient() == nil {
		t.Fatal("expected HTTP client to be configured")
	}
	if provider.Logger() == nil {
		t.Fatal("expected logger to be configured")
	}
}
