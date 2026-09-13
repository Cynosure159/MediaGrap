package httpapi

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestSourceDownload(t *testing.T) {
	build := sourceTestBuild("1.2.3")
	handler := withSourceDownload(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(418) }), build)
	for _, method := range []string{"GET", "HEAD"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, "/source", nil))
		if rec.Code != 307 || rec.Header().Get("Location") != build.SourceURL || rec.Header().Get("X-Source-SHA256") != build.SourceSHA256 || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("download: %d %v", rec.Code, rec.Header())
		}
		if method == "HEAD" && rec.Body.Len() != 0 {
			t.Fatal("HEAD returned body")
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

}

func TestSourceDownloadPreservesSourcesBookmarks(t *testing.T) {
	for _, path := range []string{"/sources", "/sources?from=bookmark"} {
		t.Run(path, func(t *testing.T) {
			handler := withSourceDownload(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RequestURI() != path {
					t.Fatalf("bookmark changed: %s", r.URL.RequestURI())
				}
				w.WriteHeader(http.StatusTeapot)
			}), BuildInfo{})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusTeapot || rec.Header().Get("Cache-Control") != "" {
				t.Fatalf("bookmark intercepted: %d %v", rec.Code, rec.Header())
			}
		})
	}
}

func sourceTestBuild(version string) BuildInfo {
	commit := strings.Repeat("a", 40)
	tag := "v" + version
	if strings.HasPrefix(version, "preview-") || strings.HasPrefix(version, "test-") {
		tag = "preview-" + commit
	}
	return BuildInfo{Version: version, Commit: commit, SourceArchitecture: runtime.GOARCH, SourceSHA256: strings.Repeat("b", 64), SourceURL: "https://github.com/Cynosure159/MediaGrap/releases/download/" + tag + "/mediagrap-source-" + version + "-" + commit + "-linux-" + runtime.GOARCH + ".tar.gz"}
}

func TestSourceUnavailable(t *testing.T) {
	for _, mutate := range []func(*BuildInfo){
		func(b *BuildInfo) { b.SourceURL = "" },
		func(b *BuildInfo) { b.SourceURL = strings.Replace(b.SourceURL, "https:", "http:", 1) },
		func(b *BuildInfo) { b.SourceURL += "?token=secret" },
		func(b *BuildInfo) { b.SourceURL += "#fragment" },
		func(b *BuildInfo) { b.SourceURL = strings.Replace(b.SourceURL, "Cynosure159", "other", 1) },
		func(b *BuildInfo) {
			b.SourceURL = strings.Replace(b.SourceURL, "download/v1.2.3", "latest/download", 1)
		},
		func(b *BuildInfo) { b.Commit = "abc" },
		func(b *BuildInfo) { b.Version = "1.2.4" },
		func(b *BuildInfo) { b.SourceSHA256 = strings.Repeat("z", 64) },
		func(b *BuildInfo) { b.SourceArchitecture = "other" },
	} {
		build := sourceTestBuild("1.2.3")
		mutate(&build)
		handler := withSourceDownload(http.NotFoundHandler(), build)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/source", nil))
		if rec.Code != 503 || rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Location") != "" {
			t.Fatalf("invalid locator advertised: %d", rec.Code)
		}
	}
}

func TestSourcePreviewAndSnapshotIdentity(t *testing.T) {
	for _, version := range []string{"preview-" + strings.Repeat("a", 40), "test-" + strings.Repeat("a", 12)} {
		if !validSourceLocator(sourceTestBuild(version)) {
			t.Fatal("valid identity rejected")
		}
	}
	for _, version := range []string{"1.0.0-rc.3", "preview-" + strings.Repeat("b", 40), "test-" + strings.Repeat("b", 12)} {
		if validSourceLocator(sourceTestBuild(version)) {
			t.Fatal("unapproved identity accepted")
		}
	}
}
