package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSourceRoutesPreserveSourcesBookmarks(t *testing.T) {
	tc := setupTestContext(t)
	settings := tc.doRequest(httptest.NewRequest(http.MethodGet, "/settings", nil), false, false)
	if settings.Code != http.StatusOK {
		t.Fatalf("SPA baseline: %d", settings.Code)
	}
	for _, path := range []string{"/sources", "/sources?from=bookmark"} {
		t.Run(path, func(t *testing.T) {
			rec := tc.doRequest(httptest.NewRequest(http.MethodGet, path, nil), false, false)
			if rec.Code != http.StatusOK || rec.Body.String() != settings.Body.String() || rec.Header().Get("Content-Type") != settings.Header().Get("Content-Type") {
				t.Fatalf("bookmark did not reach SPA: %d %v", rec.Code, rec.Header())
			}
		})
	}
}
