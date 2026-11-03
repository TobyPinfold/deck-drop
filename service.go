package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/settings"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

const (
	devicesEvent      = "devices"
	transfersEvent    = "transfers"
	filesDroppedEvent = "files-dropped"
	pairedEvent       = "paired"
	offersEvent       = "offers"
)

func init() {
	application.RegisterEvent[[]discovery.Device](devicesEvent)
	application.RegisterEvent[[]app.Transfer](transfersEvent)
	application.RegisterEvent[[]string](filesDroppedEvent)
	application.RegisterEvent[[]trust.Peer](pairedEvent)
	application.RegisterEvent[[]app.PendingOffer](offersEvent)
}

// service exposes the app to the frontend and ties it to the application
// lifecycle. Only its exported methods are callable from the frontend.
type service struct {
	app     *app.App
	desktop *application.App
}

func (service *service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	service.app.Startup(ctx)
	return nil
}

func (service *service) ServiceShutdown() error {
	service.app.Shutdown(context.Background())
	return nil
}

// Devices lists the other DeckDrop devices currently on the network.
func (service *service) Devices() []discovery.Device {
	return service.app.Devices()
}

// Transfers lists every file queued for sending so far.
func (service *service) Transfers() []app.Transfer {
	return service.app.Transfers()
}

// SendFiles queues the files at paths to be sent to the named device.
func (service *service) SendFiles(deviceName string, paths []string) error {
	return service.app.SendFiles(deviceName, paths)
}

// PauseTransfer stops a queued or in-flight transfer.
func (service *service) PauseTransfer(id int) error {
	return service.app.Pause(id)
}

// ResumeTransfer continues a paused or failed transfer.
func (service *service) ResumeTransfer(id int) error {
	return service.app.Resume(id)
}

// ReceivedFiles lists the files that have been received, newest first.
func (service *service) ReceivedFiles() ([]app.ReceivedFile, error) {
	return service.app.ReceivedFiles()
}

// ReceiveDir is the folder received files are saved to.
func (service *service) ReceiveDir() string {
	directory, err := filepath.Abs(service.app.ReceiveDir())
	if err != nil {
		return service.app.ReceiveDir()
	}
	return directory
}

// OpenReceiveFolder shows the receive folder in the system file manager.
func (service *service) OpenReceiveFolder() error {
	directory := service.ReceiveDir()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	return service.desktop.Browser.OpenFile(directory)
}

// ChooseFiles asks the user to pick files and returns their paths. It returns
// nothing if the user cancels.
func (service *service) ChooseFiles() ([]string, error) {
	return service.desktop.Dialog.OpenFile().
		SetTitle("Choose files to send").
		PromptForMultipleSelection()
}

// Settings returns the settings the app is running with.
func (service *service) Settings() settings.Settings {
	return service.app.Settings()
}

// SaveSettings applies and saves new settings. The backend restarts, so a
// taken port is reported as an error.
func (service *service) SaveSettings(updated settings.Settings) error {
	return service.app.SaveSettings(updated)
}

// ChooseFolder asks the user to pick a folder and returns its path. It
// returns an empty string if the user cancels.
func (service *service) ChooseFolder() (string, error) {
	return service.desktop.Dialog.OpenFile().
		SetTitle("Choose where received files are saved").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

// Info reports whether the backend is running and where.
func (service *service) Info() app.Info {
	return service.app.Info()
}

// StartPairing shows a one-time code another device enters to pair with this
// one.
func (service *service) StartPairing() (pairing.Code, error) {
	return service.app.StartPairing()
}

// CancelPairing discards the pairing code.
func (service *service) CancelPairing() {
	service.app.CancelPairing()
}

// PairWith pairs with the named device using the code it is showing.
func (service *service) PairWith(ctx context.Context, deviceName, code string) error {
	return service.app.PairWith(ctx, deviceName, code)
}

// Unpair revokes the pairing with the device that has fingerprint.
func (service *service) Unpair(fingerprint string) error {
	return service.app.Unpair(fingerprint)
}

// PairedDevices lists the devices this one is paired with.
func (service *service) PairedDevices() []trust.Peer {
	return service.app.PairedDevices()
}

// Offers lists the files other devices are waiting to send.
func (service *service) Offers() []app.PendingOffer {
	return service.app.Offers()
}

// RespondToOffer accepts or declines files another device wants to send.
func (service *service) RespondToOffer(id int, accept bool) error {
	return service.app.RespondToOffer(id, accept)
}

// SetAutoAccept chooses whether a paired device can send without asking.
func (service *service) SetAutoAccept(fingerprint string, enabled bool) error {
	return service.app.SetAutoAccept(fingerprint, enabled)
}
