package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
	"time"
)

// Transfer statuses.
const (
	StatusWaiting  = "waiting"
	StatusDeclined = "declined"
	StatusQueued   = "queued"
	StatusSending  = "sending"
	StatusPaused   = "paused"
	StatusDone     = "done"
	StatusFailed   = "failed"
)

// Sender uploads one file to the device at address ("host:port"), which must
// present the certificate with fingerprint (any if empty), reporting
// progress as it goes. Cancelling ctx stops the upload; sending the file again
// resumes it.
type Sender interface {
	Send(ctx context.Context, address, fingerprint, path string, progress func(sent, total int64)) error
}

// Offerer is a Sender that can ask the receiving device to accept files
// before they are sent. Senders without it send straight away.
type Offerer interface {
	Offer(ctx context.Context, address, fingerprint string, paths []string) error
}

// Transfer is one file queued to be sent to a device.
type Transfer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Device string `json:"device"`
	Status string `json:"status"`
	Error  string `json:"error"`

	// Sent and Total are in bytes; Speed is in bytes per second.
	Sent  int64   `json:"sent"`
	Total int64   `json:"total"`
	Speed float64 `json:"speed"`

	path        string
	address     string
	fingerprint string
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
	device, err := app.deviceNamed(deviceName)
	if err != nil {
		app.mu.Unlock()
		return err
	}
	address := net.JoinHostPort(device.Host, strconv.Itoa(device.Port))
	fingerprint := ""
	if app.config.Trust != nil {
		if !app.config.Trust.Has(device.Fingerprint) {
			app.mu.Unlock()
			return fmt.Errorf("pair with %q before sending files to it", deviceName)
		}
		fingerprint = device.Fingerprint
	}
	offerer, asks := app.options.Sender.(Offerer)
	status := StatusQueued
	if asks {
		status = StatusWaiting
	}
	var ids []int
	for _, path := range paths {
		ids = append(ids, len(app.transfers)+1)
		app.transfers = append(app.transfers, Transfer{
			ID:          len(app.transfers) + 1,
			Name:        filepath.Base(path),
			Device:      deviceName,
			Status:      status,
			path:        path,
			address:     address,
			fingerprint: fingerprint,
		})
	}
	startWorker := len(paths) > 0 && !asks && app.claimWorker()
	startOffer := len(paths) > 0 && asks
	if startOffer {
		app.sendersDone.Add(1)
	}
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
	if startWorker {
		go app.sendQueued()
	}
	if startOffer {
		go app.offerFiles(offerer, address, fingerprint, paths, ids)
	}
	return nil
}

// offerFiles waits for the receiving device to answer, then queues the files
// if they were accepted. The files wait without a deadline of their own, as
// the receiver decides how long to wait for its user.
func (app *App) offerFiles(offerer Offerer, address, fingerprint string, paths []string, ids []int) {
	defer app.sendersDone.Done()

	err := offerer.Offer(app.sendContext, address, fingerprint, paths)
	if err != nil && app.sendContext.Err() == nil {
		app.log.Error("failed to offer files", "device", address, "error", err)
	}

	app.mu.Lock()
	startWorker := false
	for _, id := range ids {
		index := app.indexOf(id)
		if index < 0 || app.transfers[index].Status != StatusWaiting {
			continue
		}
		switch {
		case err == nil:
			app.transfers[index].Status = StatusQueued
		case errors.Is(err, transfer.ErrDeclined):
			app.transfers[index].Status = StatusDeclined
		default:
			app.transfers[index].Status = StatusFailed
			app.transfers[index].Error = err.Error()
		}
	}
	if err == nil {
		startWorker = app.claimWorker()
	}
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
	if startWorker {
		go app.sendQueued()
	}
}

// deviceNamed must be called with app.mu held.
func (app *App) deviceNamed(deviceName string) (discovery.Device, error) {
	for _, device := range app.devices {
		if device.Name == deviceName {
			return device, nil
		}
	}
	return discovery.Device{}, fmt.Errorf("device %q was not found on the network", deviceName)
}

// Pause stops a queued or in-flight transfer. Resume continues it from what
// the device already received.
func (app *App) Pause(id int) error {
	app.mu.Lock()
	index := app.indexOf(id)
	if index < 0 {
		app.mu.Unlock()
		return fmt.Errorf("transfer %d not found", id)
	}
	transfer := &app.transfers[index]
	if transfer.Status != StatusQueued && transfer.Status != StatusSending {
		app.mu.Unlock()
		return nil
	}
	if cancel, sending := app.cancels[id]; sending && transfer.Status == StatusSending {
		cancel()
	}
	transfer.Status = StatusPaused
	transfer.Speed = 0
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
	return nil
}

// Resume queues a paused or failed transfer again.
func (app *App) Resume(id int) error {
	app.mu.Lock()
	index := app.indexOf(id)
	if index < 0 {
		app.mu.Unlock()
		return fmt.Errorf("transfer %d not found", id)
	}
	transfer := &app.transfers[index]
	if transfer.Status != StatusPaused && transfer.Status != StatusFailed {
		app.mu.Unlock()
		return nil
	}
	transfer.Status = StatusQueued
	transfer.Error = ""
	startWorker := app.claimWorker()
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
	if startWorker {
		go app.sendQueued()
	}
	return nil
}

// indexOf must be called with app.mu held.
func (app *App) indexOf(id int) int {
	return slices.IndexFunc(app.transfers, func(transfer Transfer) bool { return transfer.ID == id })
}

// claimWorker reports whether the caller must start a worker. It must be
// called with app.mu held.
func (app *App) claimWorker() bool {
	if app.sending {
		return false
	}
	app.sending = true
	app.sendersDone.Add(1)
	return true
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
		ctx, cancel := context.WithCancel(app.sendContext)
		current := app.transfers[index]
		app.cancels[current.ID] = cancel
		app.transfers[index].Status = StatusSending
		snapshot := slices.Clone(app.transfers)
		app.mu.Unlock()

		app.notifyTransfers(snapshot)

		tracker := &progressTracker{}
		err := app.options.Sender.Send(ctx, current.address, current.fingerprint, current.path, func(sent, total int64) {
			app.recordProgress(current.ID, tracker, sent, total)
		})
		if err != nil && ctx.Err() == nil {
			app.log.Error("failed to send file", "file", current.path, "device", current.Device, "error", err)
		}
		cancel()

		app.mu.Lock()
		delete(app.cancels, current.ID)
		transfer := &app.transfers[index]
		switch {
		case err == nil:
			transfer.Status = StatusDone
			transfer.Sent = transfer.Total
			transfer.Speed = 0
		case transfer.Status == StatusSending:
			transfer.Status = StatusFailed
			transfer.Error = err.Error()
			transfer.Speed = 0
		}
		snapshot = slices.Clone(app.transfers)
		app.mu.Unlock()

		app.notifyTransfers(snapshot)
	}
}

// progressTracker measures upload speed between progress reports.
type progressTracker struct {
	lastTime time.Time
	lastSent int64
	speed    float64
}

// recordProgress stores a progress report and, at most once per
// ProgressInterval, recomputes the smoothed speed and notifies the listener.
func (app *App) recordProgress(id int, tracker *progressTracker, sent, total int64) {
	now := app.now()

	app.mu.Lock()
	index := app.indexOf(id)
	if index < 0 || app.transfers[index].Status != StatusSending {
		app.mu.Unlock()
		return
	}

	if tracker.lastTime.IsZero() {
		tracker.lastTime, tracker.lastSent = now, sent
	} else if elapsed := now.Sub(tracker.lastTime); elapsed <= 0 || elapsed < app.options.ProgressInterval {
		app.mu.Unlock()
		return
	} else {
		instant := float64(sent-tracker.lastSent) / elapsed.Seconds()
		if tracker.speed == 0 {
			tracker.speed = instant
		} else {
			tracker.speed = (tracker.speed + instant) / 2
		}
		tracker.lastTime, tracker.lastSent = now, sent
	}

	app.transfers[index].Sent = sent
	app.transfers[index].Total = total
	app.transfers[index].Speed = tracker.speed
	snapshot := slices.Clone(app.transfers)
	app.mu.Unlock()

	app.notifyTransfers(snapshot)
}

func (app *App) now() time.Time {
	if app.options.Now != nil {
		return app.options.Now()
	}
	return time.Now()
}

func (app *App) notifyTransfers(snapshot []Transfer) {
	if app.options.OnTransfersChanged != nil {
		app.options.OnTransfersChanged(snapshot)
	}
}
