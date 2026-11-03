package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/discovery"
)

type sentFile struct{ address, path string }

type fakeSender struct {
	mu     sync.Mutex
	sent   []sentFile
	failed map[string]bool
}

func (sender *fakeSender) Send(_ context.Context, address, path string) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.sent = append(sender.sent, sentFile{address, path})
	if sender.failed[path] {
		return errors.New("connection refused")
	}
	return nil
}

func (sender *fakeSender) sentFiles() []sentFile {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]sentFile(nil), sender.sent...)
}

// newSendingApp returns a started App that has already discovered otherDevice.
func newSendingApp(t *testing.T, sender Sender) (*App, chan []Transfer) {
	t.Helper()

	devices := newDeviceEvents()
	transfers := make(chan []Transfer, 64)
	config := backend.DefaultConfig()
	config.Port = 0
	config.ReceiveDir = t.TempDir()
	config.DeviceName = "this-device"
	app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{
		Scanner:            &fakeScanner{devices: []discovery.Device{otherDevice}},
		ScanInterval:       time.Millisecond,
		OnDevicesChanged:   devices.record,
		Sender:             sender,
		OnTransfersChanged: func(list []Transfer) { transfers <- list },
	})
	app.Startup(context.Background())
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	devices.next(t)
	return app, transfers
}

// waitForFinished returns the first transfer list in which nothing is pending.
func waitForFinished(t *testing.T, transfers chan []Transfer) []Transfer {
	t.Helper()
	deadline := time.After(eventTimeout)
	for {
		select {
		case list := <-transfers:
			finished := true
			for _, transfer := range list {
				if transfer.Status == StatusQueued || transfer.Status == StatusSending {
					finished = false
				}
			}
			if finished {
				return list
			}
		case <-deadline:
			t.Fatal("timed out waiting for transfers to finish")
		}
	}
}

func TestSendFilesSendsEachFileToTheDevice(t *testing.T) {
	sender := &fakeSender{}
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav", "/games/b.sav"}); err != nil {
		t.Fatalf("SendFiles: %v", err)
	}

	list := waitForFinished(t, transfers)
	want := []sentFile{{"192.168.1.20:8080", "/games/a.sav"}, {"192.168.1.20:8080", "/games/b.sav"}}
	if got := sender.sentFiles(); !reflect.DeepEqual(got, want) {
		t.Errorf("sent = %v, want %v", got, want)
	}
	if len(list) != 2 || list[0].Status != StatusDone || list[1].Status != StatusDone {
		t.Errorf("transfers = %+v, want two done", list)
	}
	if list[0].Name != "a.sav" || list[0].Device != otherDevice.Name {
		t.Errorf("first transfer = %+v, want name a.sav to %s", list[0], otherDevice.Name)
	}
}

func TestSendFilesRejectsUnknownDevice(t *testing.T) {
	app, _ := newSendingApp(t, &fakeSender{})

	if err := app.SendFiles("nobody", []string{"/games/a.sav"}); err == nil {
		t.Fatal("SendFiles to an unknown device succeeded")
	}
	if got := app.Transfers(); len(got) != 0 {
		t.Errorf("transfers = %+v, want none", got)
	}
}

func TestFailedTransferIsReportedAndQueueContinues(t *testing.T) {
	sender := &fakeSender{failed: map[string]bool{"/games/a.sav": true}}
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav", "/games/b.sav"}); err != nil {
		t.Fatalf("SendFiles: %v", err)
	}

	list := waitForFinished(t, transfers)
	if list[0].Status != StatusFailed || list[0].Error == "" {
		t.Errorf("first transfer = %+v, want failed with an error", list[0])
	}
	if list[1].Status != StatusDone {
		t.Errorf("second transfer = %+v, want done", list[1])
	}
}
