package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
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

func (sender *fakeSender) Send(_ context.Context, address, _, path string, _ func(sent, total int64)) error {
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
		Now:                (&fakeClock{now: time.Unix(0, 0)}).Now,
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
				if transfer.Status == StatusWaiting || transfer.Status == StatusQueued || transfer.Status == StatusSending {
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

// scriptedSender runs a function per file so tests control progress and timing.
type scriptedSender func(ctx context.Context, path string, progress func(sent, total int64)) error

func (sender scriptedSender) Send(ctx context.Context, _, _, path string, progress func(sent, total int64)) error {
	return sender(ctx, path, progress)
}

// fakeClock advances one second every time it is read.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *fakeClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(time.Second)
	return clock.now
}

// waitFor returns the first transfer list that satisfies condition.
func waitFor(t *testing.T, transfers chan []Transfer, condition func([]Transfer) bool) []Transfer {
	t.Helper()
	deadline := time.After(eventTimeout)
	for {
		select {
		case list := <-transfers:
			if condition(list) {
				return list
			}
		case <-deadline:
			t.Fatal("timed out waiting for the expected transfer state")
		}
	}
}

func TestTransferProgressAndSpeedAreReported(t *testing.T) {
	proceed := make(chan struct{})
	sender := scriptedSender(func(_ context.Context, _ string, progress func(sent, total int64)) error {
		progress(0, 1000)
		progress(500, 1000)
		<-proceed
		return nil
	})
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatalf("SendFiles: %v", err)
	}

	list := waitFor(t, transfers, func(list []Transfer) bool { return list[0].Sent == 500 })
	if list[0].Total != 1000 || list[0].Speed != 500 {
		t.Errorf("transfer = %+v, want 500 of 1000 at 500 bytes/s", list[0])
	}
	close(proceed)
	list = waitForFinished(t, transfers)
	if list[0].Sent != 1000 || list[0].Speed != 0 {
		t.Errorf("finished transfer = %+v, want all 1000 bytes sent and no speed", list[0])
	}
}

// blockUntilCancelled behaves like a long upload that only stops when asked to.
func blockUntilCancelled(ctx context.Context, _ string, progress func(sent, total int64)) error {
	progress(100, 1000)
	<-ctx.Done()
	return ctx.Err()
}

func TestPausingAnInFlightTransferStopsItAndResumingContinuesIt(t *testing.T) {
	var attempts atomic.Int64
	sender := scriptedSender(func(ctx context.Context, path string, progress func(sent, total int64)) error {
		if attempts.Add(1) == 1 {
			return blockUntilCancelled(ctx, path, progress)
		}
		return nil
	})
	app, transfers := newSendingApp(t, sender)
	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatalf("SendFiles: %v", err)
	}
	waitFor(t, transfers, func(list []Transfer) bool { return list[0].Sent == 100 })

	if err := app.Pause(1); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	list := waitFor(t, transfers, func(list []Transfer) bool { return list[0].Status == StatusPaused })
	if list[0].Error != "" {
		t.Errorf("paused transfer has error %q, want none", list[0].Error)
	}

	if err := app.Resume(1); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	list = waitFor(t, transfers, func(list []Transfer) bool { return list[0].Status == StatusDone })
	if attempts.Load() != 2 {
		t.Errorf("transfer = %+v after %d attempts, want done after 2", list[0], attempts.Load())
	}
}

func TestPausedQueuedTransferIsSkippedUntilResumed(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	var order []string
	sender := scriptedSender(func(ctx context.Context, path string, progress func(sent, total int64)) error {
		mu.Lock()
		order = append(order, path)
		mu.Unlock()
		if path == "/games/a.sav" {
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	app, transfers := newSendingApp(t, sender)
	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav", "/games/b.sav"}); err != nil {
		t.Fatalf("SendFiles: %v", err)
	}
	waitFor(t, transfers, func(list []Transfer) bool { return list[0].Status == StatusSending })

	if err := app.Pause(2); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	close(release)
	list := waitFor(t, transfers, func(list []Transfer) bool { return list[0].Status == StatusDone })
	if list[1].Status != StatusPaused {
		t.Fatalf("second transfer = %+v, want it to stay paused", list[1])
	}

	if err := app.Resume(2); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	waitFor(t, transfers, func(list []Transfer) bool { return list[1].Status == StatusDone })
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(order, []string{"/games/a.sav", "/games/b.sav"}) {
		t.Errorf("send order = %v, want a then b", order)
	}
}

func TestPauseAndResumeRejectUnknownTransfers(t *testing.T) {
	app, _ := newSendingApp(t, &fakeSender{})

	if err := app.Pause(42); err == nil {
		t.Error("Pause of an unknown transfer succeeded")
	}
	if err := app.Resume(42); err == nil {
		t.Error("Resume of an unknown transfer succeeded")
	}
}
