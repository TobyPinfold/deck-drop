// Package app is the desktop application layer. It owns the embedded backend
// and exposes the methods the frontend can call.
package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

// App ties the backend's lifetime to the desktop window's lifetime.
type App struct {
	log        *slog.Logger
	backend    *backend.Backend
	options    Options
	selfName   string
	startError string

	mu      sync.Mutex
	devices []discovery.Device

	stopDiscovery context.CancelFunc
	discoveryDone chan struct{}
}

// Options configures optional parts of the App.
type Options struct {
	// Scanner finds other devices. Without one, no devices are discovered.
	Scanner discovery.Scanner

	// ScanInterval is the pause between scans.
	ScanInterval time.Duration

	// OnDevicesChanged is called when the list of other devices changes.
	OnDevicesChanged func([]discovery.Device)
}

// Info describes the embedded backend for the frontend.
type Info struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Error   string `json:"error"`
}

// New returns an App that will run a backend with config.
func New(logger *slog.Logger, config backend.Config, options Options) *App {
	return &App{
		log:      logger,
		backend:  backend.New(logger, config),
		options:  options,
		selfName: discovery.InstanceName(config.DeviceName),
	}
}

// Startup starts the backend and device discovery. A backend failure is
// reported through Info rather than preventing the window from opening.
func (app *App) Startup(_ context.Context) {
	if err := app.backend.Start(); err != nil {
		app.log.Error("failed to start backend", "error", err)
		app.startError = err.Error()
	}
	app.startDiscovery()
}

// Shutdown stops device discovery and the backend.
func (app *App) Shutdown(_ context.Context) {
	app.stopDiscoveryAndWait()
	if err := app.backend.Stop(); err != nil {
		app.log.Error("failed to stop backend", "error", err)
	}
}

// Info reports whether the backend is running and where.
func (app *App) Info() Info {
	return Info{
		Running: app.backend.Running(),
		Port:    app.backend.Port(),
		Error:   app.startError,
	}
}

// DefaultReceiveDir is where the desktop app saves files. A relative path
// would resolve against the working directory, which is "/" when the app is
// launched from a desktop launcher.
func DefaultReceiveDir(homeDir string) string {
	return filepath.Join(homeDir, "Downloads", "DeckDrop")
}
