package backend

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func newTestBackend(logBuffer *bytes.Buffer) *Backend {
	return New(slog.New(slog.NewTextHandler(logBuffer, nil)))
}

func TestStartAndStop(t *testing.T) {
	testBackend := newTestBackend(&bytes.Buffer{})

	if testBackend.Running() {
		t.Fatal("backend should not be running before Start")
	}
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !testBackend.Running() {
		t.Fatal("backend should be running after Start")
	}
	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if testBackend.Running() {
		t.Fatal("backend should not be running after Stop")
	}
}

func TestStartTwiceFails(t *testing.T) {
	testBackend := newTestBackend(&bytes.Buffer{})
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := testBackend.Start(); err == nil {
		t.Fatal("second Start should return an error")
	}
}

func TestStopWhenNotRunningIsNoop(t *testing.T) {
	testBackend := newTestBackend(&bytes.Buffer{})
	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop on idle backend: %v", err)
	}
}

func TestLogsStartup(t *testing.T) {
	var logBuffer bytes.Buffer
	testBackend := newTestBackend(&logBuffer)
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.Contains(logBuffer.String(), "backend started") {
		t.Fatalf("expected startup log, got %q", logBuffer.String())
	}
}
