package app

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"slices"
	"strconv"
)

// Transfer statuses.
const (
	StatusQueued  = "queued"
	StatusSending = "sending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Sender uploads one file to the device at address ("host:port").
type Sender interface {
	Send(ctx context.Context, address, path string) error
}

// Transfer is one file queued to be sent to a device.
type Transfer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Device string `json:"device"`
	Status string `json:"status"`
	Error  string `json:"error"`

	path    string
	address string
}

// Transfers returns every transfer queued since the app started.
func (app *App) Transfers() []Transfer {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.transfers)
}

// SendFiles queues the files at paths to be sent, one at a time, to the named
// device.
func (app *App) SendFiles(deviceName string, paths []string) error {
	if app.options.Sender == nil {
		return fmt.Errorf("sending files is not available")
	}

	app.mu.Lock()
	address, err := app.addressOf(deviceName)
	if err != nil {
		app.mu.Unlock()
		return err
	}
	for _, path := range paths {
		app.transfers = append(app.transfers, Transfer{
			ID:      len(app.transfers) + 1,
			Name:    filepath.Base(path),
			Device:  deviceName,
			Status:  StatusQueued,
			path:    path,
			address: address,
		})
	}
	startWorker := !app.sending && len(paths) > 0
	if startWorker {
		app.sending = true
		app.sendersDone.Add(1)
	}
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
	if startWorker {
		go app.sendQueued()
	}
	return nil
}

// addressOf must be called with app.mu held.
func (app *App) addressOf(deviceName string) (string, error) {
	for _, device := range app.devices {
		if device.Name == deviceName {
			return net.JoinHostPort(device.Host, strconv.Itoa(device.Port)), nil
		}
	}
	return "", fmt.Errorf("device %q was not found on the network", deviceName)
}

// sendQueued sends queued transfers in order until none are left.
func (app *App) sendQueued() {
	defer app.sendersDone.Done()

	for {
		app.mu.Lock()
		index := slices.IndexFunc(app.transfers, func(transfer Transfer) bool {
			return transfer.Status == StatusQueued
		})
		if index < 0 || app.sendContext.Err() != nil {
			app.sending = false
			app.mu.Unlock()
			return
		}
		app.transfers[index].Status = StatusSending
		current := app.transfers[index]
		snapshot := slices.Clone(app.transfers)
		app.mu.Unlock()

		app.notifyTransfers(snapshot)

		err := app.options.Sender.Send(app.sendContext, current.address, current.path)
		if err != nil {
			app.log.Error("failed to send file", "file", current.path, "device", current.Device, "error", err)
		}

		app.mu.Lock()
		app.transfers[index].Status = StatusDone
		if err != nil {
			app.transfers[index].Status = StatusFailed
			app.transfers[index].Error = err.Error()
		}
		snapshot = slices.Clone(app.transfers)
		app.mu.Unlock()

		app.notifyTransfers(snapshot)
	}
}

func (app *App) notifyTransfers(snapshot []Transfer) {
	if app.options.OnTransfersChanged != nil {
		app.options.OnTransfersChanged(snapshot)
	}
}
