package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

//go:embed all:frontend/dist
var assets embed.FS

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

// Info reports whether the backend is running and where.
func (service *service) Info() app.Info {
	return service.app.Info()
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	config := backend.DefaultConfig()
	config.Announcer = discovery.MDNS{}

	desktop := application.New(application.Options{
		Name:        "DeckDrop",
		Description: "Steam Deck first local network file sharing",
		Services:    []application.Service{application.NewService(&service{app: app.New(logger, config)})},
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
