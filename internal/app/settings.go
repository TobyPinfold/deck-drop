package app

import (
	"fmt"
	"strings"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/settings"
)

// Settings returns the values the app is currently running with.
func (app *App) Settings() settings.Settings {
	app.mu.Lock()
	defer app.mu.Unlock()
	return settings.Settings{
		DeviceName: app.config.DeviceName,
		ReceiveDir: app.config.ReceiveDir,
		Port:       app.config.Port,
	}
}

// SaveSettings applies new settings by restarting the backend, then persists
// them. If the backend cannot start with the new settings, the previous ones
// stay in effect and nothing is saved.
func (app *App) SaveSettings(updated settings.Settings) error {
	updated.DeviceName = strings.TrimSpace(updated.DeviceName)
	updated.ReceiveDir = strings.TrimSpace(updated.ReceiveDir)
	if err := updated.Validate(); err != nil {
		return err
	}

	app.restartMu.Lock()
	defer app.restartMu.Unlock()

	app.mu.Lock()
	previous := app.config
	app.mu.Unlock()

	next := updated.ApplyTo(previous)
	if next != previous {
		if err := app.restartBackend(previous, next); err != nil {
			return err
		}
	}

	if app.options.SettingsStore == nil {
		return nil
	}
	if err := app.options.SettingsStore.Save(updated); err != nil {
		return fmt.Errorf("settings applied but could not be saved: %w", err)
	}
	return nil
}

// restartBackend replaces the running backend with one using next. If that
// fails, it starts the backend with previous again.
func (app *App) restartBackend(previous, next backend.Config) error {
	if err := app.currentBackend().Stop(); err != nil {
		app.log.Error("failed to stop backend", "error", err)
	}

	replacement := backend.New(app.log, next)
	if err := replacement.Start(); err != nil {
		app.log.Error("failed to start backend with new settings", "error", err)
		restored := backend.New(app.log, previous)
		if restoreErr := restored.Start(); restoreErr != nil {
			app.log.Error("failed to restore backend", "error", restoreErr)
			app.install(restored, previous, restoreErr)
		} else {
			app.install(restored, previous, nil)
		}
		return fmt.Errorf("start with new settings: %w", err)
	}

	app.install(replacement, next, nil)
	return nil
}

func (app *App) install(running *backend.Backend, config backend.Config, startErr error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.backend = running
	app.config = config
	app.selfName = discovery.InstanceName(config.DeviceName)
	app.receiveDir = config.ReceiveDir
	app.startError = ""
	if startErr != nil {
		app.startError = startErr.Error()
	}
}
