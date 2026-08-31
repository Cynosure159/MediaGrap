package httpapi

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/settings"
)

//go:embed ui/fallback.html ui/dist/*
var ui embed.FS

type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

type server struct {
	logger          *slog.Logger
	db              *sql.DB
	build           BuildInfo
	auth            *auth.Service
	library         *library.Service
	metadata        *metadata.Service
	settingsService *settings.Service
}

func NewServer(logger *slog.Logger, db *sql.DB, build BuildInfo, authService *auth.Service, libraryService *library.Service, metadataService *metadata.Service, settingsService *settings.Service) http.Handler {
	application := &server{logger: logger, db: db, build: build, auth: authService, library: libraryService, metadata: metadataService, settingsService: settingsService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", application.health)
	mux.HandleFunc("GET /readyz", application.ready)
	mux.HandleFunc("GET /api/v1/system/info", application.systemInfo)
	mux.HandleFunc("GET /api/v1/system/summary", application.systemSummary)
	mux.HandleFunc("GET /api/v1/setup/status", application.setupStatus)
	mux.HandleFunc("POST /api/v1/setup", application.setup)
	mux.HandleFunc("POST /api/v1/session", application.login)
	mux.HandleFunc("DELETE /api/v1/session", application.logout)
	mux.HandleFunc("GET /api/v1/session", application.session)
	mux.HandleFunc("GET /api/v1/settings", application.settings)
	mux.HandleFunc("PUT /api/v1/settings", application.settings)
	mux.HandleFunc("GET /api/v1/sources", application.sources)
	mux.HandleFunc("POST /api/v1/sources", application.sources)
	mux.HandleFunc("POST /api/v1/sources/", application.scanSource)
	mux.HandleFunc("GET /api/v1/media", application.media)
	mux.HandleFunc("GET /api/v1/media/", application.mediaDetail)
	mux.HandleFunc("POST /api/v1/media/", application.mediaDetail)
	mux.HandleFunc("GET /api/v1/write-plans/", application.writePlan)
	mux.HandleFunc("POST /api/v1/write-plans/", application.writePlan)
	mux.HandleFunc("GET /api/v1/jobs", application.jobs)
	mux.Handle("/", application.frontend())
	return application.withRequestLogging(mux)
}

func (s *server) health(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) ready(writer http.ResponseWriter, request *http.Request) {
	context, cancel := request.Context(), func() {}
	defer cancel()
	if err := s.db.PingContext(context); err != nil {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *server) systemInfo(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"name":    "MediaGrap",
		"version": s.build.Version,
		"commit":  s.build.Commit,
		"builtAt": s.build.BuiltAt,
	})
}

func (s *server) systemSummary(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"status": "ready",
		"library": map[string]int{
			"sources": 0,
			"items":   0,
		},
		"jobs": map[string]int{
			"queued":  0,
			"running": 0,
			"failed":  0,
		},
		"phase": "foundation",
	})
}

func (s *server) frontend() http.Handler {
	uiRoot, err := fs.Sub(ui, "ui/dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(uiRoot))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, err := fs.Stat(uiRoot, "index.html"); err != nil {
			serveFallback(writer)
			return
		}
		if request.URL.Path != "/" {
			if _, err := fs.Stat(uiRoot, strings.TrimPrefix(request.URL.Path, "/")); err != nil {
				request.URL.Path = "/"
			}
		}
		files.ServeHTTP(writer, request)
	})
}

func serveFallback(writer http.ResponseWriter) {
	contents, err := ui.ReadFile("ui/fallback.html")
	if err != nil {
		http.Error(writer, "frontend is unavailable", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusServiceUnavailable)
	_, _ = writer.Write(contents)
}

func (s *server) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(writer, request)
		s.logger.Debug("HTTP request", "method", request.Method, "path", request.URL.Path, "duration", time.Since(startedAt))
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
