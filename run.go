package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	backendconfig "github.com/TobyPinfold/deck-drop/internal/backend/config"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/security"
	"github.com/TobyPinfold/deck-drop/internal/settings"
)

const usageExitCode = 2

func run() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	defaults := backend.DefaultConfig()
	if homeDir, err := os.UserHomeDir(); err == nil {
		defaults.ReceiveDir = app.DefaultReceiveDir(homeDir)
	}

	var settingsStore app.SettingsStore
	if store, storeError := openSettingsStore(); storeError == nil {
		settingsStore = store
		saved, loadError := store.Load()
		if loadError != nil {
			logger.Warn("ignoring saved settings", "error", loadError)
		}
		defaults = saved.ApplyTo(defaults)
	}

	config, err := backendconfig.LoadConfig(defaults, os.Args[1:], os.Stderr)
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

	if config.Headless {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := backend.Serve(ctx, logger, config); err != nil {
			logger.Error("backend failed", "error", err)
			os.Exit(1)
		}
		return
	}

	runDesktop(logger, config, settingsStore)
}

func openSettingsStore() (settings.Store, error) {
	path, err := settings.DefaultPath()
	if err != nil {
		return settings.Store{}, err
	}
	return settings.NewStore(path), nil
}
