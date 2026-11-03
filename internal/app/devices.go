package app

import (
	"context"
	"slices"

	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

// Devices returns the other DeckDrop devices currently on the network.
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
	others := make([]discovery.Device, 0, len(found))
	for _, device := range found {
		if device.Name != app.selfName {
			others = append(others, device)
		}
	}

	app.mu.Lock()
	app.devices = others
	app.mu.Unlock()

	if app.options.OnDevicesChanged != nil {
		app.options.OnDevicesChanged(slices.Clone(others))
	}
}
