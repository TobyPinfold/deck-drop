package backend

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func newTestBackend(buf *bytes.Buffer) *Backend {
	return New(slog.New(slog.NewTextHandler(buf, nil)))
}

func TestStartAndStop(t *testing.T) {
	b := newTestBackend(&bytes.Buffer{})

	if b.Running() {
		t.Fatal("backend should not be running before Start")
	}
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !b.Running() {
		t.Fatal("backend should be running after Start")
	}
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if b.Running() {
		t.Fatal("backend should not be running after Stop")
	}
}

func TestStartTwiceFails(t *testing.T) {
	b := newTestBackend(&bytes.Buffer{})
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := b.Start(); err == nil {
		t.Fatal("second Start should return an error")
	}
}

func TestStopWhenNotRunningIsNoop(t *testing.T) {
	b := newTestBackend(&bytes.Buffer{})
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop on idle backend: %v", err)
	}
}

func TestLogsStartup(t *testing.T) {
	var buf bytes.Buffer
	b := newTestBackend(&buf)
	if err := b.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.Contains(buf.String(), "backend started") {
		t.Fatalf("expected startup log, got %q", buf.String())
	}
}
