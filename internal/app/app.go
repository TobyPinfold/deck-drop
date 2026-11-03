// Package app is the desktop application layer. It owns the embedded backend
// and exposes the methods the frontend can call.
package app

import (
	"context"
	"log/slog"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

// App ties the backend's lifetime to the desktop window's lifetime.
type App struct {
	log     *slog.Logger
	backend *backend.Backend

	startError string
}

// Info describes the embedded backend for the frontend.
type Info struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Error   string `json:"error"`
}

// New returns an App that will run a backend with config.
func New(logger *slog.Logger, config backend.Config) *App {
	return &App{log: logger, backend: backend.New(logger, config)}
}

// Startup starts the backend. A failure is reported through Info rather than
// preventing the window from opening.
func (app *App) Startup(_ context.Context) {
	if err := app.backend.Start(); err != nil {
		app.log.Error("failed to start backend", "error", err)
		app.startError = err.Error()
	}
}

// Shutdown stops the backend.
func (app *App) Shutdown(_ context.Context) {
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
