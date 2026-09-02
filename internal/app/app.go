package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
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
	config Config
	logger *slog.Logger
	db     *sql.DB
	server *http.Server
	worker *library.Service
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
	libraryService.SetFFprobePath(config.FFprobePath)
	settingsService := settings.NewService(db, settings.Defaults{TMDbAPIKey: config.TMDbAPIKey, FanartTVAPIKey: config.FanartTVAPIKey, TMDbLanguage: config.TMDbLanguage, FallbackLanguage: config.FallbackLanguage, OutboundProxy: config.OutboundProxy, NoProxy: config.NoProxy, MediaRoots: config.MediaRoots})
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
	if err := metadataService.ConfigureProviders(currentSettings.TMDbAPIKey, currentSettings.FanartTVAPIKey, currentSettings.TMDbLanguage, currentSettings.FallbackLanguage, currentSettings.OutboundProxy, currentSettings.NoProxy); err != nil {
		db.Close()
		return nil, err
	}
	handler := httpapi.NewServer(logger, db, httpapi.BuildInfo(build), auth.NewService(db), libraryService, metadataService, settingsService, httpapi.RuntimePaths{ConfigDir: config.ConfigDir, CacheDir: config.CacheDir})
	return &Application{
		config: config,
		logger: logger,
		db:     db,
		server: &http.Server{
			Addr:              config.Listen,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		worker: libraryService,
	}, nil
}

func (a *Application) Run(ctx context.Context) error {
	go a.worker.RunWorker(ctx)
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
