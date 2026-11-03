package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

func TestSavedSettingsAreLoadedAgain(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "settings.json"))
	want := Settings{DeviceName: "living-room", ReceiveDir: "/data/drops", Port: 9090}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}

func TestLoadWithoutAFileReturnsNoSettings(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "settings.json"))

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (Settings{}) {
		t.Fatalf("Load = %+v, want empty settings", got)
	}
}

func TestLoadRejectsACorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStore(path).Load(); err == nil {
		t.Fatal("Load should fail on a corrupt file")
	}
}

func TestHiddenSettingIsApplied(t *testing.T) {
	if !(Settings{Hidden: true}).ApplyTo(backend.Config{}).Hidden {
		t.Error("Hidden was not applied")
	}
	if (Settings{}).ApplyTo(backend.Config{Hidden: true}).Hidden {
		t.Error("Hidden was not cleared")
	}
}

func TestApplyToOverridesOnlyTheSettingsThatAreSet(t *testing.T) {
	defaults := backend.Config{DeviceName: "default-name", ReceiveDir: "/default", Port: 8080}

	got := Settings{DeviceName: "custom"}.ApplyTo(defaults)

	want := backend.Config{DeviceName: "custom", ReceiveDir: "/default", Port: 8080}
	if got != want {
		t.Fatalf("ApplyTo = %+v, want %+v", got, want)
	}
}

func TestValidate(t *testing.T) {
	valid := Settings{DeviceName: "deck", ReceiveDir: "/data", Port: 8080}
	tests := map[string]func(Settings) Settings{
		"empty name":      func(settings Settings) Settings { settings.DeviceName = "  "; return settings },
		"empty directory": func(settings Settings) Settings { settings.ReceiveDir = ""; return settings },
		"port too low":    func(settings Settings) Settings { settings.Port = 0; return settings },
		"port too high":   func(settings Settings) Settings { settings.Port = 65536; return settings },
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) = %v", err)
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			if err := change(valid).Validate(); err == nil {
				t.Fatal("Validate should fail")
			}
		})
	}
}
