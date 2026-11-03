package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	config := backend.DefaultConfig()
	config.Port = 0
	config.ReceiveDir = t.TempDir()
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{})
}

func TestStartupStartsTheBackendAndShutdownStopsIt(t *testing.T) {
	app := newTestApp(t)

	app.Startup(context.Background())
	info := app.Info()
	if !info.Running || info.Port == 0 {
		t.Fatalf("Info after Startup = %+v, want running on a port", info)
	}

	response, err := http.Get("http://" + app.backend.Addr() + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	_ = response.Body.Close()

	app.Shutdown(context.Background())
	if app.Info().Running {
		t.Fatal("backend still running after Shutdown")
	}
}

func TestStartupReportsAFailureToStartTheBackend(t *testing.T) {
	app := newTestApp(t)
	app.Startup(context.Background())
	defer app.Shutdown(context.Background())

	second := New(slog.New(slog.NewTextHandler(io.Discard, nil)), backend.Config{Port: app.Info().Port, ReceiveDir: t.TempDir()}, Options{})
	second.Startup(context.Background())

	if second.Info().Running {
		t.Fatal("second app should not be running on a taken port")
	}
	if second.Info().Error == "" {
		t.Fatal("Info.Error should explain why the backend failed to start")
	}
}

func TestDefaultReceiveDirIsInsideTheHomeDownloads(t *testing.T) {
	got := DefaultReceiveDir("/home/deck")

	want := filepath.Join("/home/deck", "Downloads", "DeckDrop")
	if got != want {
		t.Fatalf("DefaultReceiveDir = %q, want %q", got, want)
	}
}
