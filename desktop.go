package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/TobyPinfold/deck-drop/internal/app"
	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/history"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

const (
	scanDuration     = 3 * time.Second
	scanInterval     = 2 * time.Second
	progressInterval = 200 * time.Millisecond
)

func runDesktop(logger *slog.Logger, config backend.Config, settingsStore app.SettingsStore) {
	var desktop *application.App
	deckDrop := app.New(logger, config, app.Options{
		Scanner:      discovery.MDNSScanner{Duration: scanDuration},
		ScanInterval: scanInterval,
		OnDevicesChanged: func(devices []discovery.Device) {
			desktop.Event.Emit(devicesEvent, devices)
		},
		SettingsStore:    settingsStore,
		HistoryStore:     openHistoryStore(),
		Sender:           transfer.Sender{Secure: true},
		ProgressInterval: progressInterval,
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

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		desktop.Event.Emit(filesDroppedEvent, event.Context().DroppedFiles())
	})

	if err := desktop.Run(); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

// openHistoryStore returns nil when there is nowhere to keep the history, in
// which case only files received during this run are listed.
func openHistoryStore() app.HistoryStore {
	path, err := history.DefaultPath()
	if err != nil {
		return nil
	}
	return history.NewStore(path)
}
