package app

import "time"

const defaultPairingModeDuration = 2 * time.Minute

// PairingMode says whether unpaired devices are currently listed.
type PairingMode struct {
	Enabled   bool      `json:"enabled"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// GetPairingMode reports whether pairing mode is on.
func (app *App) GetPairingMode() PairingMode {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.pairingMode()
}

// SetPairingMode turns pairing mode on, which lists nearby unpaired devices
// so they can be paired, or off. It turns itself off after a couple of
// minutes.
func (app *App) SetPairingMode(enabled bool) PairingMode {
	app.mu.Lock()
	if app.pairingTimer != nil {
		app.pairingTimer.Stop()
		app.pairingTimer = nil
	}
	app.pairingEnd = time.Time{}
	if enabled {
		duration := app.options.PairingModeDuration
		if duration <= 0 {
			duration = defaultPairingModeDuration
		}
		app.pairingEnd = app.now().Add(duration)
		app.pairingTimer = time.AfterFunc(duration, func() { app.SetPairingMode(false) })
	}
	mode := app.pairingMode()
	app.mu.Unlock()

	app.refreshDevices()
	if app.options.OnPairingModeChanged != nil {
		app.options.OnPairingModeChanged(mode)
	}
	return mode
}

func (app *App) stopPairingTimer() {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.pairingTimer != nil {
		app.pairingTimer.Stop()
		app.pairingTimer = nil
	}
}

// pairingModeActive must be called with app.mu held.
func (app *App) pairingModeActive() bool {
	return !app.pairingEnd.IsZero()
}

// pairingMode must be called with app.mu held.
func (app *App) pairingMode() PairingMode {
	return PairingMode{Enabled: app.pairingModeActive(), ExpiresAt: app.pairingEnd}
}
