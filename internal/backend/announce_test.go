package backend

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type fakeAnnouncer struct {
	err          error
	instances    []string
	ports        []int
	fingerprints []string
	closed       int
}

func (announcer *fakeAnnouncer) Announce(instance string, port int, fingerprint string) (io.Closer, error) {
	if announcer.err != nil {
		return nil, announcer.err
	}
	announcer.instances = append(announcer.instances, instance)
	announcer.ports = append(announcer.ports, port)
	announcer.fingerprints = append(announcer.fingerprints, fingerprint)
	return closerFunc(func() error {
		announcer.closed++
		return nil
	}), nil
}

type closerFunc func() error

func (closer closerFunc) Close() error { return closer() }

func newAnnouncingBackend(t *testing.T, logBuffer *bytes.Buffer, announcer Announcer) *Backend {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(logBuffer, nil))
	return New(logger, Config{Port: 0, ReceiveDir: t.TempDir(), DeviceName: "steam-deck", Announcer: announcer})
}

func TestStartAnnouncesDeviceOnListeningPort(t *testing.T) {
	announcer := &fakeAnnouncer{}
	testBackend := newAnnouncingBackend(t, &bytes.Buffer{}, announcer)
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	if len(announcer.instances) != 1 || announcer.instances[0] != "steam-deck" {
		t.Fatalf("announced instances = %v, want [steam-deck]", announcer.instances)
	}
	if announcer.ports[0] != testBackend.Port() || testBackend.Port() == 0 {
		t.Fatalf("announced port = %d, listening on %d", announcer.ports[0], testBackend.Port())
	}
}

func TestStopEndsAnnouncement(t *testing.T) {
	announcer := &fakeAnnouncer{}
	testBackend := newAnnouncingBackend(t, &bytes.Buffer{}, announcer)
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if announcer.closed != 1 {
		t.Fatalf("announcement closed %d times, want 1", announcer.closed)
	}
}

func TestStartSucceedsWhenAnnouncingFails(t *testing.T) {
	var logBuffer bytes.Buffer
	announcer := &fakeAnnouncer{err: errors.New("no multicast")}
	testBackend := newAnnouncingBackend(t, &logBuffer, announcer)

	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start should tolerate announce failure: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	if !testBackend.Running() {
		t.Fatal("backend should be running")
	}
	if !strings.Contains(logBuffer.String(), "no multicast") {
		t.Fatalf("expected announce failure to be logged, got %q", logBuffer.String())
	}
}

func TestStartWithoutAnnouncerDoesNotAnnounce(t *testing.T) {
	testBackend := newAnnouncingBackend(t, &bytes.Buffer{}, nil)

	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	if !testBackend.Running() {
		t.Fatal("backend should be running")
	}
}

func TestHiddenDeviceIsNotAnnounced(t *testing.T) {
	announcer := &fakeAnnouncer{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	testBackend := New(logger, Config{Port: 0, ReceiveDir: t.TempDir(), DeviceName: "steam-deck", Announcer: announcer, Hidden: true})
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	if len(announcer.instances) != 0 {
		t.Errorf("a hidden device announced %v", announcer.instances)
	}
}
