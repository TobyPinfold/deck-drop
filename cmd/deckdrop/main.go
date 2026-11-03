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
	"github.com/TobyPinfold/deck-drop/internal/security"
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

	config, err = security.Setup(config)
	if err != nil {
		logger.Error("cannot set up device security", "error", err)
		os.Exit(1)
	}
	logger.Info("device identity ready", "fingerprint", config.Identity.Fingerprint())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := backend.Serve(ctx, logger, config); err != nil {
		logger.Error("backend failed", "error", err)
		os.Exit(1)
	}
}
