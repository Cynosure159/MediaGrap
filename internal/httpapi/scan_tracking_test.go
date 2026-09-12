package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/auth"
)

func TestScanTrackingEndpointsBoundedAuthenticatedAndRedacted(t *testing.T) {
	db := testDatabase(t)
	server := testServer(db)
	session, err := auth.NewService(db).Setup(t.Context(), "admin", "long-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO jobs(id,kind,state,payload) VALUES(1,'scan','running','{"secret":"must-not-leak"}')`); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/jobs/1", "/api/v1/jobs?activeScans=true&after=0"} {
		for _, authorized := range []bool{false, true} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if authorized {
				req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: session.Token})
			}
			res := httptest.NewRecorder()
			server.ServeHTTP(res, req)
			expected := http.StatusUnauthorized
			if authorized {
				expected = http.StatusOK
			}
			if res.Code != expected || strings.Contains(res.Body.String(), "must-not-leak") || strings.Contains(res.Body.String(), "payload") {
				t.Fatalf("%s %d %s", path, res.Code, res.Body.String())
			}
			if authorized && !strings.Contains(res.Body.String(), `"state":"running"`) {
				t.Fatal(res.Body.String())
			}
		}
	}
	for _, tc := range []struct {
		path   string
		status int
	}{{"/api/v1/jobs?activeScans=true&after=-1", 400}, {"/api/v1/jobs?activeScans=true&after=bad", 400}, {"/api/v1/jobs/0", 400}, {"/api/v1/jobs/9999", 404}} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: session.Token})
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("%s %d", tc.path, res.Code)
		}
	}
}
