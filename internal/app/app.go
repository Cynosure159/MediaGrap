package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/platform/database"
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

	handler := httpapi.NewServer(logger, db, httpapi.BuildInfo(build))
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
	}, nil
}

func (a *Application) Run(ctx context.Context) error {
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
