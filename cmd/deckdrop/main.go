package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	b := backend.New(logger)

	if err := b.Start(); err != nil {
		logger.Error("failed to start backend", "error", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if err := b.Stop(); err != nil {
		logger.Error("failed to stop backend", "error", err)
		os.Exit(1)
	}
}
