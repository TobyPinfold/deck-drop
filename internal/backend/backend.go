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
func (b *Backend) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return errors.New("backend already running")
	}
	b.running = true
	b.log.Info("backend started")
	return nil
}

// Stop shuts the backend down. It is a no-op if not running.
func (b *Backend) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}
	b.running = false
	b.log.Info("backend stopped")
	return nil
}

// Running reports whether the backend is currently started.
func (b *Backend) Running() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}
