package app

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

const eventTimeout = 5 * time.Second

type fakeScanner struct {
	devices []discovery.Device
	scans   atomic.Int64
}

func (scanner *fakeScanner) Scan(context.Context) ([]discovery.Device, error) {
	scanner.scans.Add(1)
	return scanner.devices, nil
}

type deviceEvents struct {
	changed chan []discovery.Device
}

func newDeviceEvents() *deviceEvents {
	return &deviceEvents{changed: make(chan []discovery.Device, 16)}
}

func (events *deviceEvents) record(devices []discovery.Device) { events.changed <- devices }

func (events *deviceEvents) next(t *testing.T) []discovery.Device {
	t.Helper()
	select {
	case devices := <-events.changed:
		return devices
	case <-time.After(eventTimeout):
		t.Fatal("timed out waiting for a device list")
		return nil
	}
}

func newDiscoveringApp(t *testing.T, scanner discovery.Scanner, events *deviceEvents) *App {
	t.Helper()

	config := backend.DefaultConfig()
	config.Port = 0
	config.ReceiveDir = t.TempDir()
	config.DeviceName = "this-device"
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{
		Scanner:          scanner,
		ScanInterval:     time.Millisecond,
		OnDevicesChanged: events.record,
	})
}

var otherDevice = discovery.Device{Name: "steam-deck", Host: "192.168.1.20", Port: 8080}

func TestDiscoveredDevicesAreReportedAndListed(t *testing.T) {
	events := newDeviceEvents()
	app := newDiscoveringApp(t, &fakeScanner{devices: []discovery.Device{otherDevice}}, events)

	app.Startup(context.Background())
	defer app.Shutdown(context.Background())

	want := []discovery.Device{otherDevice}
	if got := events.next(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("reported devices = %v, want %v", got, want)
	}
	if got := app.Devices(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Devices() = %v, want %v", got, want)
	}
}

func TestThisDeviceIsNotListed(t *testing.T) {
	events := newDeviceEvents()
	self := discovery.Device{Name: "this-device", Host: "192.168.1.10", Port: 8080}
	app := newDiscoveringApp(t, &fakeScanner{devices: []discovery.Device{self, otherDevice}}, events)

	app.Startup(context.Background())
	defer app.Shutdown(context.Background())

	want := []discovery.Device{otherDevice}
	if got := events.next(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("reported devices = %v, want %v", got, want)
	}
}

func TestShutdownStopsScanning(t *testing.T) {
	scanner := &fakeScanner{}
	app := newDiscoveringApp(t, scanner, newDeviceEvents())

	app.Startup(context.Background())
	app.Shutdown(context.Background())

	scansAtShutdown := scanner.scans.Load()
	time.Sleep(50 * time.Millisecond)
	if got := scanner.scans.Load(); got != scansAtShutdown {
		t.Fatalf("scans continued after Shutdown: %d -> %d", scansAtShutdown, got)
	}
}
