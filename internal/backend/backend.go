// Package backend contains the DeckDrop file transfer service that the
// desktop app embeds.
package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/transfer"
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

	announcement io.Closer
	receiver     *transfer.Receiver
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

	receiver, err := transfer.NewReceiver(backend.config.ReceiveDir)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", backend.config.Port))
	if err != nil {
		_ = receiver.Close()
		return fmt.Errorf("listen on port %d: %w", backend.config.Port, err)
	}

	server := &http.Server{
		Handler:           newRouter(receiver, backend.log, backend.config.ReceiveDir, backend.config.maxChunkSize()),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	backend.server = server
	backend.listener = listener
	backend.receiver = receiver
	backend.served = make(chan struct{})

	go backend.serve(server, listener, backend.served)
	backend.announce()

	backend.log.Info("backend started", "addr", listener.Addr().String(), "dir", backend.config.ReceiveDir)
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

	backend.stopAnnouncing()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := backend.server.Shutdown(ctx)
	<-backend.served // Serve closes the listener on return, so the port is free afterwards
	closeErr := backend.receiver.Close()
	backend.server = nil
	backend.listener = nil
	backend.served = nil
	backend.receiver = nil
	if err != nil {
		return fmt.Errorf("shut down http server: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close receive directory: %w", closeErr)
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

// Port returns the port the server is listening on, or zero when stopped.
func (backend *Backend) Port() int {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	if backend.listener == nil {
		return 0
	}
	return backend.listener.Addr().(*net.TCPAddr).Port
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

// announce advertises the device. A failure is logged rather than returned
// because file receiving still works without discovery.
func (backend *Backend) announce() {
	if backend.config.Announcer == nil {
		return
	}

	port := backend.listener.Addr().(*net.TCPAddr).Port
	announcement, err := backend.config.Announcer.Announce(backend.config.DeviceName, port)
	if err != nil {
		backend.log.Warn("failed to announce device on the network", "error", err)
		return
	}
	backend.announcement = announcement
}

func (backend *Backend) stopAnnouncing() {
	if backend.announcement == nil {
		return
	}
	if err := backend.announcement.Close(); err != nil {
		backend.log.Warn("failed to stop announcing device", "error", err)
	}
	backend.announcement = nil
}
