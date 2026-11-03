package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	config := backend.DefaultConfig()
	config.Announcer = discovery.MDNS{}
	backendService := backend.New(logger, config)

	if err := backendService.Start(); err != nil {
		logger.Error("failed to start backend", "error", err)
		os.Exit(1)
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignals

	if err := backendService.Stop(); err != nil {
		logger.Error("failed to stop backend", "error", err)
		os.Exit(1)
	}
}
