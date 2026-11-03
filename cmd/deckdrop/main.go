package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	backendconfig "github.com/TobyPinfold/deck-drop/internal/backend/config"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

const usageExitCode = 2

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	config, err := backendconfig.LoadConfig(backend.DefaultConfig(), os.Args[1:], os.Stderr)
	if errors.Is(err, backendconfig.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(usageExitCode)
	}
	config.Announcer = discovery.MDNS{}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := backend.Serve(ctx, logger, config); err != nil {
		logger.Error("backend failed", "error", err)
		os.Exit(1)
	}
}
