package httpapi

import (
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

// Only the release builder sets these; there is no runtime URL override or proxy.
var sourceURL, sourceSHA256, sourceArchitecture string

var sourceCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var sourceHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var sourceVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func validSourceLocator(build BuildInfo) bool {
	if !sourceCommitPattern.MatchString(build.Commit) || !sourceHashPattern.MatchString(build.SourceSHA256) || build.SourceArchitecture != runtime.GOARCH || (build.SourceArchitecture != "amd64" && build.SourceArchitecture != "arm64") {
		return false
	}
	tag := "v" + build.Version
	if !sourceVersionPattern.MatchString(build.Version) {
		if build.Version != "preview-"+build.Commit && build.Version != "test-"+build.Commit[:12] {
			return false
		}
		tag = "preview-" + build.Commit
	}
	asset := "mediagrap-source-" + build.Version + "-" + build.Commit + "-linux-" + build.SourceArchitecture + ".tar.gz"
	return build.SourceURL == "https://github.com/Cynosure159/MediaGrap/releases/download/"+tag+"/"+asset
}

// Intercept before ServeMux canonicalization so traversal cannot reach a SPA fallback.
func withSourceDownload(next http.Handler, build BuildInfo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /sources is an existing SPA bookmark, not a source-download suffix.
		if !strings.HasPrefix(r.URL.Path, "/source") || (r.URL.Path == "/sources" && r.URL.EscapedPath() == "/sources") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.Path != "/source" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.EscapedPath() != "/source" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !validSourceLocator(build) {
			http.Error(w, "Same-build public source locator unavailable in this build", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-Source-SHA256", build.SourceSHA256)
		http.Redirect(w, r, build.SourceURL, http.StatusTemporaryRedirect)
	})
}
