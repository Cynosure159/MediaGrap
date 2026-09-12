package httpapi

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed all:distribution
var distribution embed.FS

// The archive is build input, never a runtime path or a proxy destination.
// Intercept before ServeMux canonicalization so traversal cannot reach a SPA fallback.
func withSourceDownload(next http.Handler, files fs.FS, build BuildInfo) http.Handler {
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
		var manifest struct {
			Version string
			Commit  string
			SHA256  string
		}
		data, err := fs.ReadFile(files, "distribution/manifest.json")
		if err != nil || json.Unmarshal(data, &manifest) != nil || manifest.Version != build.Version || manifest.Commit != build.Commit || len(manifest.SHA256) != 64 {
			http.Error(w, "Same-build source archive unavailable in this build", http.StatusServiceUnavailable)
			return
		}
		archive, err := files.Open("distribution/source.tar.gz")
		if err != nil {
			http.Error(w, "Same-build source archive unavailable in this build", http.StatusServiceUnavailable)
			return
		}
		defer archive.Close()
		seeker, ok := archive.(io.ReadSeeker)
		if !ok {
			http.Error(w, "Source archive unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="mediagrap-source.tar.gz"`)
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		w.Header().Set("ETag", `"`+manifest.SHA256+`"`)
		http.ServeContent(w, r, "mediagrap-source.tar.gz", time.Time{}, seeker)
	})
}
