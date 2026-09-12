package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSourceDownload(t *testing.T) {
	build := BuildInfo{Version: "1.2.3", Commit: "abc"}
	manifest, _ := json.Marshal(map[string]string{"version": build.Version, "commit": build.Commit, "sha256": strings.Repeat("a", 64)})
	files := fstest.MapFS{
		"distribution/manifest.json": &fstest.MapFile{Data: manifest},
		"distribution/source.tar.gz": &fstest.MapFile{Data: []byte("synthetic immutable archive")},
	}
	handler := withSourceDownload(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(418) }), files, build)
	for _, method := range []string{"GET", "HEAD"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, "/source", nil))
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/gzip" || !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment;") || rec.Header().Get("Cache-Control") != "public, max-age=0, must-revalidate" {
			t.Fatalf("download: %d %v", rec.Code, rec.Header())
		}
		if method == "HEAD" && rec.Body.Len() != 0 {
			t.Fatal("HEAD returned body")
		}
		if method == "GET" && rec.Body.String() != "synthetic immutable archive" {
			t.Fatal("archive changed")
		}
	}
	for _, path := range []string{"/source/../config", "/source/", "/source.tar.gz", "/source?path=/config", "/source?", "/%73ource", "/source%2fconfig", "/sources/../source", "/sources/", "/%73ources"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 404 {
			t.Fatalf("%s: %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/source", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET, HEAD" {
		t.Fatal("method not rejected")
	}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/source", nil)
	req.Header.Set("If-None-Match", `"`+strings.Repeat("a", 64)+`"`)
	handler.ServeHTTP(rec, req)
	if rec.Code != 304 {
		t.Fatal("ETag not honored")
	}
}

func TestSourceDownloadPreservesSourcesBookmarks(t *testing.T) {
	for _, path := range []string{"/sources", "/sources?from=bookmark"} {
		t.Run(path, func(t *testing.T) {
			handler := withSourceDownload(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RequestURI() != path {
					t.Fatalf("bookmark changed: %s", r.URL.RequestURI())
				}
				w.WriteHeader(http.StatusTeapot)
			}), fstest.MapFS{}, BuildInfo{})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusTeapot || rec.Header().Get("Cache-Control") != "" {
				t.Fatalf("bookmark intercepted: %d %v", rec.Code, rec.Header())
			}
		})
	}
}

func TestSourceUnavailable(t *testing.T) {
	for _, files := range []fstest.MapFS{{}, {"distribution/manifest.json": &fstest.MapFile{Data: []byte(`{"version":"other","commit":"abc"}`)}}} {
		handler := withSourceDownload(http.NotFoundHandler(), files, BuildInfo{Version: "1.2.3", Commit: "abc"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/source", nil))
		if rec.Code != 503 || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing/mismatched source must not be advertised")
		}
	}
}
