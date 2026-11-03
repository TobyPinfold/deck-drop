package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

func TestFilesAreOnlySentToPairedDevices(t *testing.T) {
	store, err := trust.Open("")
	if err != nil {
		t.Fatal(err)
	}
	device := discovery.Device{Name: "steam-deck", Host: "192.168.1.20", Port: 8080, Fingerprint: "deck-fp"}
	events := newDeviceEvents()
	config := backend.DefaultConfig()
	config.Port = 0
	config.ReceiveDir = t.TempDir()
	config.DeviceName = "this-device"
	config.Trust = store
	sender := &fakeSender{}
	app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{
		Scanner:          &fakeScanner{devices: []discovery.Device{device}},
		ScanInterval:     time.Millisecond,
		OnDevicesChanged: events.record,
		Sender:           sender,
	})
	app.Startup(context.Background())
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	events.next(t)

	if err := app.SendFiles(device.Name, []string{"/games/a.sav"}); err == nil {
		t.Fatal("SendFiles() to an unpaired device error = nil, want an error")
	}

	if err := store.Add(trust.Peer{Fingerprint: "deck-fp", Name: "steam-deck"}); err != nil {
		t.Fatal(err)
	}
	if err := app.SendFiles(device.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatalf("SendFiles() to a paired device error = %v", err)
	}

	if err := app.Unpair("deck-fp"); err != nil {
		t.Fatal(err)
	}
	if len(app.PairedDevices()) != 0 {
		t.Error("PairedDevices() is not empty after Unpair")
	}
}
