package httpapi_test

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Match the server's compile-time UI inputs, including Go-only checkouts.
//
//go:embed ui/fallback.html ui/dist/*
var sourceRoutesUI embed.FS

func TestSourceRoutesPreserveSourcesBookmarks(t *testing.T) {
	wantStatus := http.StatusOK
	wantBody, err := sourceRoutesUI.ReadFile("ui/dist/index.html")
	if errors.Is(err, fs.ErrNotExist) {
		wantStatus = http.StatusServiceUnavailable
		wantBody, err = sourceRoutesUI.ReadFile("ui/fallback.html")
	}
	if err != nil {
		t.Fatal(err)
	}
	tc := setupTestContext(t)
	settings := tc.doRequest(httptest.NewRequest(http.MethodGet, "/settings", nil), false, false)
	if settings.Code != wantStatus || settings.Body.String() != string(wantBody) || settings.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("SPA baseline: %d %v, want status %d and embedded HTML", settings.Code, settings.Header(), wantStatus)
	}
	for _, path := range []string{"/sources", "/sources?from=bookmark"} {
		t.Run(path, func(t *testing.T) {
			rec := tc.doRequest(httptest.NewRequest(http.MethodGet, path, nil), false, false)
			if rec.Code != settings.Code || rec.Body.String() != settings.Body.String() || rec.Header().Get("Content-Type") != settings.Header().Get("Content-Type") {
				t.Fatalf("bookmark did not reach SPA: %d %v", rec.Code, rec.Header())
			}
		})
	}
}
