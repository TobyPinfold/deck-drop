package app

import (
	"slices"
)

// Receiving records how much of a file from the device with fingerprint has
// arrived. It implements httpapi.ProgressListener.
func (app *App) Receiving(fingerprint, name string, received, total int64) {
	now := app.now()

	app.mu.Lock()
	index := app.incomingIndex(fingerprint, name)
	if index < 0 {
		app.transfers = append(app.transfers, Transfer{
			ID:          len(app.transfers) + 1,
			Direction:   DirectionReceive,
			Name:        name,
			Device:      app.deviceNameFor(fingerprint),
			fingerprint: fingerprint,
		})
		index = len(app.transfers) - 1
	}
	transfer := &app.transfers[index]
	finished := total > 0 && received >= total
	if !finished && !transfer.lastReport.IsZero() && now.Sub(transfer.lastReport) < app.options.ProgressInterval {
		app.mu.Unlock()
		return
	}

	if elapsed := now.Sub(transfer.lastReport); !transfer.lastReport.IsZero() && elapsed > 0 {
		instant := float64(received-transfer.lastReceived) / elapsed.Seconds()
		if transfer.Speed == 0 {
			transfer.Speed = instant
		} else {
			transfer.Speed = (transfer.Speed + instant) / 2
		}
	}
	transfer.lastReport, transfer.lastReceived = now, received
	transfer.Sent, transfer.Total = received, total
	transfer.Status = StatusReceiving
	if finished {
		transfer.Status = StatusDone
		transfer.Speed = 0
	}
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
}

// ReceiveStopped records that an upload ended before the file was complete,
// for example because the sender paused. Sending again resumes it.
func (app *App) ReceiveStopped(fingerprint, name string) {
	app.mu.Lock()
	index := app.incomingIndex(fingerprint, name)
	if index < 0 || app.transfers[index].Status != StatusReceiving {
		app.mu.Unlock()
		return
	}
	app.transfers[index].Status = StatusInterrupted
	app.transfers[index].Speed = 0
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
}

// incomingIndex finds the unfinished download of name from a device. It must
// be called with app.mu held.
func (app *App) incomingIndex(fingerprint, name string) int {
	return slices.IndexFunc(app.transfers, func(transfer Transfer) bool {
		return transfer.Direction == DirectionReceive && transfer.Name == name &&
			transfer.fingerprint == fingerprint && transfer.Status != StatusDone
	})
}

// deviceNameFor is the paired name for fingerprint. It must be called with
// app.mu held.
func (app *App) deviceNameFor(fingerprint string) string {
	if app.config.Trust != nil {
		if peer, ok := app.config.Trust.Get(fingerprint); ok {
			return peer.Name
		}
	}
	return "another device"
}
