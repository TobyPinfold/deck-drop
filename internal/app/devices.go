package app

import (
	"context"
	"slices"

	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

// Devices returns the other DeckDrop devices to show: paired ones, plus every
// nearby one while pairing mode is on.
func (app *App) Devices() []discovery.Device {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.devices)
}

func (app *App) startDiscovery() {
	if app.options.Scanner == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	app.stopDiscovery = cancel
	app.discoveryDone = make(chan struct{})

	go func() {
		defer close(app.discoveryDone)
		discovery.Watch(ctx, app.options.Scanner, app.options.ScanInterval, app.handleDevices)
	}()
}

func (app *App) stopDiscoveryAndWait() {
	if app.stopDiscovery == nil {
		return
	}
	app.stopDiscovery()
	<-app.discoveryDone
	app.stopDiscovery = nil
}

// handleDevices drops this device from the list, stores it and notifies the
// listener.
func (app *App) handleDevices(found []discovery.Device) {
	app.mu.Lock()
	app.found = found
	app.mu.Unlock()
	app.refreshDevices()
}

// refreshDevices recomputes the visible devices from what was found and
// notifies the listener if they changed. With a trust store, unpaired devices
// are hidden unless pairing mode is on.
func (app *App) refreshDevices() {
	app.mu.Lock()
	showAll := app.config.Trust == nil || app.pairingModeActive()
	visible := make([]discovery.Device, 0, len(app.found))
	for _, device := range app.found {
		if device.Name == app.selfName {
			continue
		}
		if showAll || app.config.Trust.Has(device.Fingerprint) {
			visible = append(visible, device)
		}
	}
	changed := !slices.Equal(visible, app.devices) || app.devices == nil
	app.devices = visible
	app.mu.Unlock()

	if changed && app.options.OnDevicesChanged != nil {
		app.options.OnDevicesChanged(slices.Clone(visible))
	}
}
