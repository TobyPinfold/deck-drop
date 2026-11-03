// Package backend contains the DeckDrop file transfer service that the
// desktop app embeds.
package backend

import (
	"errors"
	"log/slog"
	"sync"
)

// Backend is the embeddable transfer service.
type Backend struct {
	log     *slog.Logger
	mu      sync.Mutex
	running bool
}

// New returns a stopped Backend that logs to logger.
func New(logger *slog.Logger) *Backend {
	return &Backend{log: logger}
}

// Start brings the backend up. It returns an error if already running.
func (backend *Backend) Start() error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.running {
		return errors.New("backend already running")
	}
	backend.running = true
	backend.log.Info("backend started")
	return nil
}

// Stop shuts the backend down. It is a no-op if not running.
func (backend *Backend) Stop() error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if !backend.running {
		return nil
	}
	backend.running = false
	backend.log.Info("backend stopped")
	return nil
}

// Running reports whether the backend is currently started.
func (backend *Backend) Running() bool {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.running
}
