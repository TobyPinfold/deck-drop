// Package backend contains the DeckDrop file transfer service that the
// desktop app embeds.
package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// Backend is the embeddable transfer service.
type Backend struct {
	log    *slog.Logger
	config Config

	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	served   chan struct{}
}

// New returns a stopped Backend that logs to logger.
func New(logger *slog.Logger, config Config) *Backend {
	return &Backend{log: logger, config: config}
}

// Start begins serving HTTP. It returns an error if already running or if
// the configured port cannot be bound.
func (backend *Backend) Start() error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.server != nil {
		return errors.New("backend already running")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", backend.config.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", backend.config.Port, err)
	}

	server := &http.Server{
		Handler:           newRouter(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	backend.server = server
	backend.listener = listener
	backend.served = make(chan struct{})

	go backend.serve(server, listener, backend.served)

	backend.log.Info("backend started", "addr", listener.Addr().String())
	return nil
}

func (backend *Backend) serve(server *http.Server, listener net.Listener, served chan<- struct{}) {
	defer close(served)

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		backend.log.Error("http server failed", "error", err)
	}
}

// Stop gracefully shuts the server down, waiting for in-flight requests.
// It is a no-op if not running.
func (backend *Backend) Stop() error {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := backend.server.Shutdown(ctx)
	<-backend.served // Serve closes the listener on return, so the port is free afterwards
	backend.server = nil
	backend.listener = nil
	backend.served = nil
	if err != nil {
		return fmt.Errorf("shut down http server: %w", err)
	}

	backend.log.Info("backend stopped")
	return nil
}

// Running reports whether the backend is currently started.
func (backend *Backend) Running() bool {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.server != nil
}

// Addr returns the address the server is listening on, or an empty string
// when stopped.
func (backend *Backend) Addr() string {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.listener == nil {
		return ""
	}
	return backend.listener.Addr().String()
}
