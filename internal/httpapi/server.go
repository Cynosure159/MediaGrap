package httpapi

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"github.com/mediagrap/mediagrap/internal/automation"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/tokens"
	"github.com/mediagrap/mediagrap/internal/webhooks"
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

type RuntimePaths struct {
	Automation *automation.Service
	ConfigDir  string
	CacheDir   string
	Webhooks   *webhooks.Service
	MCP        http.Handler
}

type server struct {
	logger          *slog.Logger
	db              *sql.DB
	build           BuildInfo
	auth            AuthService
	library         LibraryService
	metadata        MetadataService
	settingsService SettingsService
	runtime         RuntimePaths
	webhooks        *webhooks.Service
	tokens          *tokens.Service
}

func NewServer(
	logger *slog.Logger,
	db *sql.DB,
	build BuildInfo,
	authService AuthService,
	libraryService LibraryService,
	metadataService MetadataService,
	settingsService SettingsService,
	runtime ...RuntimePaths,
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
	if len(runtime) > 0 {
		application.runtime = runtime[0]
	}

	application.webhooks = application.runtime.Webhooks
	if application.webhooks == nil {
		application.webhooks = webhooks.New(db, webhooks.Config{}, logger)
	}
	application.tokens = tokens.New(db)
	mux := http.NewServeMux()
	application.integrationRoutes(mux)
	if application.runtime.MCP != nil {
		mux.Handle("/mcp", application.runtime.MCP)
	}

	// Health & System
	mux.HandleFunc("GET /healthz", application.health)
	mux.HandleFunc("GET /readyz", application.ready)
	mux.HandleFunc("GET /api/v1/system/info", application.systemInfo)
	mux.HandleFunc("GET /api/v1/system/summary", application.systemSummary)
	mux.HandleFunc("GET /api/v1/operations/status", application.operationsStatus)

	// Auth & Setup
	mux.HandleFunc("GET /api/v1/setup/status", application.setupStatus)
	mux.HandleFunc("POST /api/v1/setup", application.setup)
	mux.HandleFunc("GET /api/v1/session", application.session)
	mux.HandleFunc("POST /api/v1/session", application.login)
	mux.HandleFunc("DELETE /api/v1/session", application.logout)

	// Settings
	mux.HandleFunc("GET /api/v1/settings", application.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", application.updateSettings)
	mux.HandleFunc("POST /api/v1/settings/connection-tests", application.testConnection)

	// Sources
	mux.HandleFunc("GET /api/v1/sources", application.listSources)
	mux.HandleFunc("POST /api/v1/sources", application.createSource)
	mux.HandleFunc("DELETE /api/v1/sources/{id}", application.deleteSource)
	mux.HandleFunc("POST /api/v1/sources/{id}/scans", application.scanSource)
	mux.HandleFunc("PUT /api/v1/sources/{id}/policy", application.updateSourcePolicy)

	// Media (Movies)
	mux.HandleFunc("GET /api/v1/media", application.listMedia)
	mux.HandleFunc("GET /api/v1/media/{id}", application.getMedia)
	mux.HandleFunc("GET /api/v1/media/{id}/inspection", application.getMediaInspection)
	mux.HandleFunc("POST /api/v1/media/{id}/naming-preview", application.previewMediaNaming)
	mux.HandleFunc("GET /api/v1/media/{id}/candidates", application.getMediaCandidates)
	mux.HandleFunc("GET /api/v1/media/{id}/artwork-candidates", application.getMediaArtworkCandidates)
	mux.HandleFunc("POST /api/v1/media/{id}/artwork-candidates", application.scrapeMediaArtworkCandidates)
	mux.HandleFunc("GET /api/v1/media/{id}/artwork-preview/{candidate}", application.getMediaArtworkPreview)
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
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/artwork-candidates", application.getTVArtworkCandidates)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/artwork-candidates", application.scrapeTVArtworkCandidates)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/artwork-preview/{candidate}", application.getTVArtworkPreview)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/artwork-plans", application.previewTVArtworkPlan)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/poster", application.getTVShowPoster)
	mux.HandleFunc("GET /api/v1/tv/shows/{id}/artwork/{asset}", application.getTVArtwork)

	// Plans (NFO & Artwork Safe Writes)
	mux.HandleFunc("GET /api/v1/write-plans/{id}", application.getWritePlan)
	mux.HandleFunc("POST /api/v1/write-plans/{id}/apply", application.applyWritePlan)
	mux.HandleFunc("GET /api/v1/artwork-plans/{id}", application.getArtworkPlan)
	mux.HandleFunc("POST /api/v1/artwork-plans/{id}/apply", application.applyArtworkPlan)
	mux.HandleFunc("GET /api/v1/tv/artwork-plans/{id}", application.getTVArtworkPlan)
	mux.HandleFunc("POST /api/v1/tv/artwork-plans/{id}/apply", application.applyTVArtworkPlan)

	// Rename Plans
	mux.HandleFunc("POST /api/v1/media/{id}/rename-plans", application.previewMediaRenamePlan)
	mux.HandleFunc("POST /api/v1/tv/shows/{id}/rename-plans", application.previewTVRenamePlan)
	mux.HandleFunc("GET /api/v1/rename-plans/{id}", application.getRenamePlan)
	mux.HandleFunc("POST /api/v1/rename-plans/{id}/apply", application.applyRenamePlan)

	// Jobs
	mux.HandleFunc("GET /api/v1/jobs", application.listJobs)
	mux.HandleFunc("GET /api/v1/jobs/{id}", application.getJob)
	mux.HandleFunc("GET /api/v1/jobs/events", application.streamJobEvents)
	mux.HandleFunc("POST /api/v1/jobs/{id}/cancel", application.cancelJob)
	mux.HandleFunc("POST /api/v1/jobs/{id}/retry", application.retryJob)
	mux.HandleFunc("GET /api/v1/audit-entries", application.listAuditEntries)

	// Frontend SPA
	mux.Handle("/", application.frontend())

	return application.withRequestLogging(mux)
}

func (s *server) health(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) ready(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	var one int
	if err := s.db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		stats := database.Stats(s.db)
		s.logger.Warn("database readiness check failed", "error", err, "db_open_connections", stats.OpenConnections, "db_in_use", stats.InUse, "db_wait_count", stats.WaitCount, "db_saturated", stats.Saturated)
		writeJSON(writer, http.StatusServiceUnavailable, map[string]any{"status": "not_ready", "database": stats})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"status": "ready", "database": database.Stats(s.db)})
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
	err := s.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM sources),
		(SELECT COUNT(*) FROM media_items WHERE missing=0),
		(SELECT COUNT(*) FROM tv_shows),
		(SELECT COUNT(*) FROM jobs WHERE state='queued'),
		(SELECT COUNT(*) FROM jobs WHERE state='running'),
		(SELECT COUNT(*) FROM jobs WHERE state='failed')`).Scan(&sourceCount, &mediaCount, &showCount, &queuedJobs, &runningJobs, &failedJobs)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "database_unavailable", "Database query did not complete")
		return
	}

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
		if request.Method == http.MethodGet && request.URL.Path == "/api/v1/system/summary" {
			ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
			defer cancel()
			request = request.WithContext(ctx)
		}
		next.ServeHTTP(writer, request)
		duration := time.Since(startedAt)
		if errors.Is(request.Context().Err(), context.DeadlineExceeded) {
			stats := database.Stats(s.db)
			s.logger.Warn("HTTP request timed out", "method", request.Method, "path", request.URL.Path, "duration_ms", duration.Milliseconds(), "db_open_connections", stats.OpenConnections, "db_in_use", stats.InUse, "db_wait_count", stats.WaitCount, "db_saturated", stats.Saturated)
			return
		}
		s.logger.Debug("HTTP request", "method", request.Method, "path", request.URL.Path, "duration", duration)
	})
}
