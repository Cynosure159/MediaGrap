package httpapi

import (
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
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
	auth            AuthService
	library         LibraryService
	metadata        MetadataService
	settingsService SettingsService
}

func NewServer(
	logger *slog.Logger,
	db *sql.DB,
	build BuildInfo,
	authService AuthService,
	libraryService LibraryService,
	metadataService MetadataService,
	settingsService SettingsService,
) http.Handler {
	application := &server{
		logger:          logger,
		db:              db,
		build:           build,
		auth:            authService,
		library:         libraryService,
		metadata:        metadataService,
		settingsService: settingsService,
	}

	mux := http.NewServeMux()

	// Health & System
	mux.HandleFunc("GET /healthz", application.health)
	mux.HandleFunc("GET /readyz", application.ready)
	mux.HandleFunc("GET /api/v1/system/info", application.systemInfo)
	mux.HandleFunc("GET /api/v1/system/summary", application.systemSummary)

	// Auth & Setup
	mux.HandleFunc("GET /api/v1/setup/status", application.setupStatus)
	mux.HandleFunc("POST /api/v1/setup", application.setup)
	mux.HandleFunc("GET /api/v1/session", application.session)
	mux.HandleFunc("POST /api/v1/session", application.login)
	mux.HandleFunc("DELETE /api/v1/session", application.logout)

	// Settings
	mux.HandleFunc("GET /api/v1/settings", application.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", application.updateSettings)

	// Sources
	mux.HandleFunc("GET /api/v1/sources", application.listSources)
	mux.HandleFunc("POST /api/v1/sources", application.createSource)
	mux.HandleFunc("DELETE /api/v1/sources/{id}", application.deleteSource)
	mux.HandleFunc("POST /api/v1/sources/{id}/scans", application.scanSource)

	// Media (Movies)
	mux.HandleFunc("GET /api/v1/media", application.listMedia)
	mux.HandleFunc("GET /api/v1/media/{id}", application.getMedia)
	mux.HandleFunc("GET /api/v1/media/{id}/candidates", application.getMediaCandidates)
	mux.HandleFunc("POST /api/v1/media/{id}/select", application.selectMediaCandidate)
	mux.HandleFunc("POST /api/v1/media/{id}/metadata", application.saveMediaMetadata)
	mux.HandleFunc("POST /api/v1/media/{id}/write-plans", application.previewMediaWritePlan)
	mux.HandleFunc("POST /api/v1/media/{id}/artwork-plans", application.previewMediaArtworkPlan)
	mux.HandleFunc("GET /api/v1/media/{id}/local-artwork/{kind}", application.getMediaLocalArtwork)
	mux.HandleFunc("GET /api/v1/media/{id}/artwork/{asset}", application.getMediaArtwork)

	// TV Shows & Episodes
	mux.HandleFunc("GET /api/v1/tv/shows", application.listTVShows)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}", application.getTVShow)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/candidates", application.getTVShowCandidates)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/select", application.selectTVShowCandidate)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/seasons/{season}/scrape", application.scrapeTVSeason)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/seasons/{season}/episodes/{episode}/scrape", application.scrapeTVEpisode)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/nfo-plans", application.previewTVNFOPlans)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/nfo", application.getTVNFORaw)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/poster", application.getTVShowPoster)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/artwork/{asset}", application.getTVArtwork)

	// Plans (NFO & Artwork Safe Writes)
	mux.HandleFunc("GET /api/v1/write-plans/{id}", application.getWritePlan)
	mux.HandleFunc("POST /api/v1/write-plans/{id}/apply", application.applyWritePlan)
	mux.HandleFunc("GET /api/v1/artwork-plans/{id}", application.getArtworkPlan)
	mux.HandleFunc("POST /api/v1/artwork-plans/{id}/apply", application.applyArtworkPlan)

	// Jobs
	mux.HandleFunc("GET /api/v1/jobs", application.listJobs)

	// Frontend SPA
	mux.Handle("/", application.frontend())

	return application.withRequestLogging(mux)
}

func (s *server) health(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) ready(writer http.ResponseWriter, request *http.Request) {
	if err := s.db.PingContext(request.Context()); err != nil {
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
	var sourceCount, mediaCount, showCount int
	var queuedJobs, runningJobs, failedJobs int
	ctx := request.Context()
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources`).Scan(&sourceCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items WHERE missing=0`).Scan(&mediaCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tv_shows`).Scan(&showCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE state='queued'`).Scan(&queuedJobs)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE state='running'`).Scan(&runningJobs)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs WHERE state='failed'`).Scan(&failedJobs)

	writeJSON(writer, http.StatusOK, map[string]any{
		"status": "ready",
		"library": map[string]int{
			"sources": sourceCount,
			"items":   mediaCount,
			"shows":   showCount,
		},
		"jobs": map[string]int{
			"queued":  queuedJobs,
			"running": runningJobs,
			"failed":  failedJobs,
		},
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
