package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	config := backend.DefaultConfig()
	config.Announcer = discovery.MDNS{}
	deckDrop := app.New(logger, config)

	err := wails.Run(&options.App{
		Title:            "DeckDrop",
		Width:            1024,
		Height:           768,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        deckDrop.Startup,
		OnShutdown:       deckDrop.Shutdown,
		Bind:             []interface{}{deckDrop},
	})
	if err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}
