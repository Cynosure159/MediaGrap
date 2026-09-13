package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mediagrap/mediagrap/internal/mcp"
	"github.com/mediagrap/mediagrap/internal/tokens"
	"github.com/mediagrap/mediagrap/internal/webhooks"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/automation"
	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/providers/fanart"
	"github.com/mediagrap/mediagrap/internal/settings"
)

type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

type Application struct {
	automation *automation.Service
	config     Config
	logger     *slog.Logger
	db         *sql.DB
	server     *http.Server
	worker     *library.Service
	webhooks   *webhooks.Service
	tokens     *tokens.Service
}

func New(config Config, logger *slog.Logger, build BuildInfo) (*Application, error) {
	if err := config.EnsureDirectories(); err != nil {
		return nil, err
	}

	db, err := database.Open(config.DatabasePath())
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}

	libraryService := library.NewService(db, config.MediaRoots)
	libraryService.SetLogger(logger)
	libraryService.SetOutboxLimit(config.IntegrationBacklogLimit)
	libraryService.SetFFprobePath(config.FFprobePath)
	settingsService := settings.NewService(db, settings.Defaults{TMDbAPIKey: config.TMDbAPIKey, FanartTVAPIKey: config.FanartTVAPIKey, FanartTVPersonalAPIKey: config.FanartTVPersonalAPIKey, TMDbLanguage: config.TMDbLanguage, FallbackLanguage: config.FallbackLanguage, OutboundProxy: config.OutboundProxy, NoProxy: config.NoProxy, MediaRoots: config.MediaRoots})
	currentSettings, err := settingsService.Current(context.Background())
	if err != nil {
		db.Close()
		return nil, err
	}
	outbound, err := metadata.NewOutboundClient(currentSettings.OutboundProxy, currentSettings.NoProxy)
	if err != nil {
		db.Close()
		return nil, err
	}
	metadataService := metadata.NewService(db, metadata.NewTMDb(logger, outbound, currentSettings.TMDbAPIKey))
	metadataService.SetFanartProvider(fanart.NewClient(logger, outbound, currentSettings.FanartTVAPIKey))
	metadataService.SetJobService(libraryService)
	libraryService.SetMetadataHydrator(metadataService)
	if err := metadataService.ConfigureProviders(currentSettings.TMDbAPIKey, currentSettings.FanartTVAPIKey, currentSettings.FanartTVPersonalAPIKey, currentSettings.TMDbLanguage, currentSettings.FallbackLanguage, currentSettings.OutboundProxy, currentSettings.NoProxy); err != nil {
		db.Close()
		return nil, err
	}
	webhookService := webhooks.New(db, config.Webhooks, logger)
	tokenService := tokens.New(db)
	automationService := automation.New(db, tokenService, libraryService, metadataService)
	automationService.SetBacklogLimit(config.IntegrationBacklogLimit)
	if err := automationService.Recover(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	mcpHandler := mcp.New(tokenService, libraryService, config.MCPOrigins, logger)
	mcpHandler.SetAutomation(automationService)
	handler := httpapi.NewServer(logger, db, httpapi.BuildInfo{Version: build.Version, Commit: build.Commit, BuiltAt: build.BuiltAt}, auth.NewService(db), libraryService, metadataService, settingsService, httpapi.RuntimePaths{SecureSessionCookie: config.SecureSessionCookie, ConfigDir: config.ConfigDir, CacheDir: config.CacheDir, Webhooks: webhookService, MCP: mcpHandler, Automation: automationService})
	return &Application{
		config:     config,
		automation: automationService,
		logger:     logger,
		db:         db,
		server: &http.Server{
			Addr:              config.Listen,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		worker:   libraryService,
		webhooks: webhookService, tokens: tokenService,
	}, nil
}

func (a *Application) Run(ctx context.Context) error {
	workerCtx, stopWorkers := context.WithCancel(ctx)
	var workers sync.WaitGroup
	workers.Go(func() { a.worker.RunWorker(workerCtx) })
	workers.Go(func() { a.webhooks.Run(workerCtx) })
	workers.Go(func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if err := a.automation.Cleanup(workerCtx); err != nil {
					a.logger.Error("automation retention failed")
				}
				if err := a.tokens.Cleanup(workerCtx); err != nil {
					a.logger.Error("MCP audit retention failed")
				}
			}
		}
	})
	defer func() { stopWorkers(); workers.Wait() }()
	errorChannel := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server listening", "address", a.config.Listen)
		errorChannel <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := a.server.Shutdown(shutdownContext)
		stopWorkers()
		workers.Wait()
		closeErr := a.db.Close()
		return errors.Join(err, closeErr)
	case err := <-errorChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func CheckDatabase(ctx context.Context, databasePath string) error {
	db, err := database.Open(databasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check: %w", err)
	}
	return nil
}
