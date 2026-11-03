package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

const (
	usageExitCode = 2

	scanDuration = 3 * time.Second
	scanInterval = 2 * time.Second

	devicesEvent      = "devices"
	transfersEvent    = "transfers"
	filesDroppedEvent = "files-dropped"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Registering the event type gives the frontend typed bindings for it.
	application.RegisterEvent[[]discovery.Device](devicesEvent)
	application.RegisterEvent[[]app.Transfer](transfersEvent)
	application.RegisterEvent[[]string](filesDroppedEvent)
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

// ChooseFiles asks the user to pick files and returns their paths. It returns
// nothing if the user cancels.
func (service *service) ChooseFiles() ([]string, error) {
	return service.desktop.Dialog.OpenFile().
		SetTitle("Choose files to send").
		PromptForMultipleSelection()
}

// Info reports whether the backend is running and where.
func (service *service) Info() app.Info {
	return service.app.Info()
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	defaults := backend.DefaultConfig()
	if homeDir, err := os.UserHomeDir(); err == nil {
		defaults.ReceiveDir = app.DefaultReceiveDir(homeDir)
	}
	config, err := backend.LoadConfig(defaults, os.Args[1:], os.Stderr)
	if errors.Is(err, backend.ErrHelp) {
		return
	}
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(usageExitCode)
	}
	config.Announcer = discovery.MDNS{}

	if config.Headless {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := backend.Serve(ctx, logger, config); err != nil {
			logger.Error("backend failed", "error", err)
			os.Exit(1)
		}
		return
	}

	var desktop *application.App
	deckDrop := app.New(logger, config, app.Options{
		Scanner:      discovery.MDNSScanner{Duration: scanDuration},
		ScanInterval: scanInterval,
		OnDevicesChanged: func(devices []discovery.Device) {
			desktop.Event.Emit(devicesEvent, devices)
		},
		Sender: transfer.Sender{},
		OnTransfersChanged: func(transfers []app.Transfer) {
			desktop.Event.Emit(transfersEvent, transfers)
		},
	})
	deckDropService := &service{app: deckDrop}

	desktop = application.New(application.Options{
		Name:        "DeckDrop",
		Description: "Steam Deck first local network file sharing",
		Services:    []application.Service{application.NewService(deckDropService)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	deckDropService.desktop = desktop

	window := desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "DeckDrop",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		EnableFileDrop:   true,
	})

	// The frontend knows which device is selected, so it decides where
	// dropped files go.
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		desktop.Event.Emit(filesDroppedEvent, event.Context().DroppedFiles())
	})

	if err := desktop.Run(); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}
