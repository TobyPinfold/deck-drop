package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/settings"
)

type recordingStore struct {
	saved []settings.Settings
	err   error
}

func (store *recordingStore) Save(saved settings.Settings) error {
	if store.err != nil {
		return store.err
	}
	store.saved = append(store.saved, saved)
	return nil
}

func newStartedApp(t *testing.T, store SettingsStore) *App {
	t.Helper()

	config := backend.DefaultConfig()
	config.Port = freePort(t)
	config.ReceiveDir = t.TempDir()
	app := New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{SettingsStore: store})
	app.Startup(context.Background())
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	return app
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func TestSettingsReportsTheCurrentValues(t *testing.T) {
	app := newStartedApp(t, nil)

	got := app.Settings()

	if got.DeviceName == "" || got.ReceiveDir == "" || got.Port != app.Info().Port {
		t.Fatalf("Settings = %+v, want the running configuration", got)
	}
}

func TestSaveSettingsRestartsTheBackendWithTheNewValuesAndPersistsThem(t *testing.T) {
	store := &recordingStore{}
	app := newStartedApp(t, store)
	want := settings.Settings{DeviceName: "renamed", ReceiveDir: filepath.Join(t.TempDir(), "drops"), Port: freePort(t)}

	if err := app.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	if info := app.Info(); !info.Running || info.Port != want.Port {
		t.Fatalf("Info = %+v, want running on port %d", info, want.Port)
	}
	if got := app.Settings(); got != want {
		t.Fatalf("Settings = %+v, want %+v", got, want)
	}
	if got := app.ReceiveDir(); got != want.ReceiveDir {
		t.Fatalf("ReceiveDir = %q, want %q", got, want.ReceiveDir)
	}
	if len(store.saved) != 1 || store.saved[0] != want {
		t.Fatalf("saved = %+v, want [%+v]", store.saved, want)
	}
}

func TestSaveSettingsKeepsTheOldConfigurationWhenThePortIsTaken(t *testing.T) {
	store := &recordingStore{}
	app := newStartedApp(t, store)
	before := app.Settings()

	taken, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	changed := before
	changed.Port = taken.Addr().(*net.TCPAddr).Port

	if err := app.SaveSettings(changed); err == nil {
		t.Fatal("SaveSettings should fail when the port is taken")
	}

	if info := app.Info(); !info.Running || info.Port != before.Port {
		t.Fatalf("Info = %+v, want still running on port %d", info, before.Port)
	}
	if got := app.Settings(); got != before {
		t.Fatalf("Settings = %+v, want unchanged %+v", got, before)
	}
	if len(store.saved) != 0 {
		t.Fatalf("saved = %+v, want nothing saved", store.saved)
	}
}

func TestSaveSettingsRejectsInvalidSettings(t *testing.T) {
	app := newStartedApp(t, &recordingStore{})
	invalid := app.Settings()
	invalid.DeviceName = ""

	if err := app.SaveSettings(invalid); err == nil {
		t.Fatal("SaveSettings should reject an empty device name")
	}
}

func TestSaveSettingsReportsAFailureToPersist(t *testing.T) {
	app := newStartedApp(t, &recordingStore{err: errors.New("disk full")})
	changed := app.Settings()
	changed.DeviceName = "renamed"

	if err := app.SaveSettings(changed); err == nil {
		t.Fatal("SaveSettings should report that the settings were not saved")
	}
}
