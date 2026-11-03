package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

const (
	scanDuration = 3 * time.Second
	scanInterval = 2 * time.Second

	devicesEvent = "devices"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Registering the event type gives the frontend typed bindings for it.
	application.RegisterEvent[[]discovery.Device](devicesEvent)
}

// service exposes the app to the frontend and ties it to the application
// lifecycle. Only its exported methods are callable from the frontend.
type service struct {
	app *app.App
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

// Info reports whether the backend is running and where.
func (service *service) Info() app.Info {
	return service.app.Info()
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	config := backend.DefaultConfig()
	config.Announcer = discovery.MDNS{}
	if homeDir, err := os.UserHomeDir(); err == nil {
		config.ReceiveDir = app.DefaultReceiveDir(homeDir)
	}

	var desktop *application.App
	deckDrop := app.New(logger, config, app.Options{
		Scanner:      discovery.MDNSScanner{Duration: scanDuration},
		ScanInterval: scanInterval,
		OnDevicesChanged: func(devices []discovery.Device) {
			desktop.Event.Emit(devicesEvent, devices)
		},
	})

	desktop = application.New(application.Options{
		Name:        "DeckDrop",
		Description: "Steam Deck first local network file sharing",
		Services:    []application.Service{application.NewService(&service{app: deckDrop})},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "DeckDrop",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	if err := desktop.Run(); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}
