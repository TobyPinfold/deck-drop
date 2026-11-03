package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

var errPairingUnavailable = errors.New("pairing is not available")

// StartPairing shows a one-time code that another device enters to pair with
// this one. Starting again replaces the code.
func (app *App) StartPairing() (pairing.Code, error) {
	config := app.currentConfig()
	if config.Pairing == nil {
		return pairing.Code{}, errPairingUnavailable
	}
	return config.Pairing.NewCode()
}

// CancelPairing discards the active pairing code.
func (app *App) CancelPairing() {
	if pairer := app.currentConfig().Pairing; pairer != nil {
		pairer.Cancel()
	}
}

// PairWith pairs with the named device using the code it is showing.
func (app *App) PairWith(ctx context.Context, deviceName, code string) error {
	app.mu.Lock()
	config := app.config
	device, err := app.deviceNamed(deviceName)
	app.mu.Unlock()
	if config.Identity == nil || config.Trust == nil {
		return errPairingUnavailable
	}
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", device.Host, device.Port)
	peer, err := pairing.Pair(ctx, config.Identity, config.DeviceName, address, code)
	if err != nil {
		return err
	}
	return config.Trust.Add(peer)
}

// Unpair revokes the pairing with the device that has fingerprint. The
// device can no longer send files here, and files are no longer sent to it
// without pairing again.
func (app *App) Unpair(fingerprint string) error {
	store := app.currentConfig().Trust
	if store == nil {
		return errPairingUnavailable
	}
	return store.Remove(fingerprint)
}

// PairedDevices lists the devices this one is paired with.
func (app *App) PairedDevices() []trust.Peer {
	store := app.currentConfig().Trust
	if store == nil {
		return nil
	}
	return store.List()
}

func (app *App) currentConfig() backend.Config {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.config
}
