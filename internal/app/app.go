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
	"github.com/TobyPinfold/deck-drop/internal/settings"
)

// App ties the backend's lifetime to the desktop window's lifetime.
type App struct {
	log     *slog.Logger
	options Options

	// restartMu serialises settings changes, which restart the backend.
	restartMu sync.Mutex

	mu         sync.Mutex
	backend    *backend.Backend
	config     backend.Config
	selfName   string
	receiveDir string
	startError string
	devices    []discovery.Device
	transfers  []Transfer
	sending    bool
	cancels    map[int]context.CancelFunc

	sendContext context.Context
	stopSending context.CancelFunc
	sendersDone sync.WaitGroup

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

	// Sender uploads files to other devices. Without one, sending is disabled.
	Sender Sender

	// OnTransfersChanged is called whenever a transfer is added or changes
	// status.
	OnTransfersChanged func([]Transfer)

	// ProgressInterval is the shortest time between progress updates for one
	// transfer. Zero reports every update.
	ProgressInterval time.Duration

	// SettingsStore persists changed settings. Without one, changes only last
	// until the app closes.
	SettingsStore SettingsStore

	// Now returns the current time. It defaults to time.Now.
	Now func() time.Time
}

// SettingsStore saves the settings a user changed.
type SettingsStore interface {
	Save(settings.Settings) error
}

// Info describes the embedded backend for the frontend.
type Info struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Error   string `json:"error"`
}

// New returns an App that will run a backend with config.
func New(logger *slog.Logger, config backend.Config, options Options) *App {
	sendContext, stopSending := context.WithCancel(context.Background())
	return &App{
		log:         logger,
		backend:     backend.New(logger, config),
		config:      config,
		options:     options,
		selfName:    discovery.InstanceName(config.DeviceName),
		receiveDir:  config.ReceiveDir,
		sendContext: sendContext,
		stopSending: stopSending,
		cancels:     make(map[int]context.CancelFunc),
	}
}

// Startup starts the backend and device discovery. A backend failure is
// reported through Info rather than preventing the window from opening.
func (app *App) Startup(_ context.Context) {
	if err := app.currentBackend().Start(); err != nil {
		app.log.Error("failed to start backend", "error", err)
		app.setStartError(err)
	}
	app.startDiscovery()
}

// Shutdown stops device discovery, in-flight sends and the backend.
func (app *App) Shutdown(_ context.Context) {
	app.stopDiscoveryAndWait()
	app.stopSending()
	app.sendersDone.Wait()
	app.restartMu.Lock()
	defer app.restartMu.Unlock()
	if err := app.currentBackend().Stop(); err != nil {
		app.log.Error("failed to stop backend", "error", err)
	}
}

// Info reports whether the backend is running and where.
func (app *App) Info() Info {
	running := app.currentBackend()
	app.mu.Lock()
	defer app.mu.Unlock()
	return Info{
		Running: running.Running(),
		Port:    running.Port(),
		Error:   app.startError,
	}
}

func (app *App) currentBackend() *backend.Backend {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.backend
}

func (app *App) setStartError(err error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if err == nil {
		app.startError = ""
		return
	}
	app.startError = err.Error()
}

// DefaultReceiveDir is where the desktop app saves files. A relative path
// would resolve against the working directory, which is "/" when the app is
// launched from a desktop launcher.
func DefaultReceiveDir(homeDir string) string {
	return filepath.Join(homeDir, "Downloads", "DeckDrop")
}
