package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mediagrap/mediagrap/internal/app"
)

var (
	version = "dev"
	commit  = "none"
	builtAt = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	config, err := app.LoadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	logger := app.NewLogger(config.LogFormat, config.LogLevel)
	application, err := app.New(config, logger, app.BuildInfo{
		Version: version,
		Commit:  commit,
		BuiltAt: builtAt,
	})
	if err != nil {
		logger.Error("application startup failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func runHealthcheck() {
	config, err := app.LoadConfig(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := app.CheckReadiness(ctx, config.Listen); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
