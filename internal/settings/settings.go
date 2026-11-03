// Package settings persists the settings a user can change from the app.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

const (
	maxPort      = 65535
	directoryDir = "DeckDrop"
	fileName     = "settings.json"
)

// Settings are the values a user can change.
type Settings struct {
	DeviceName string `json:"deviceName"`
	ReceiveDir string `json:"receiveDir"`
	Port       int    `json:"port"`

	// Hidden stops this device announcing itself on the network.
	Hidden bool `json:"hidden"`
}

// Validate reports why the settings cannot be used.
func (settings Settings) Validate() error {
	if strings.TrimSpace(settings.DeviceName) == "" {
		return errors.New("device name must not be empty")
	}
	if strings.TrimSpace(settings.ReceiveDir) == "" {
		return errors.New("receive folder must not be empty")
	}
	if settings.Port < 1 || settings.Port > maxPort {
		return fmt.Errorf("port must be between 1 and %d, got %d", maxPort, settings.Port)
	}
	return nil
}

// ApplyTo returns config with every setting that has a value applied.
func (settings Settings) ApplyTo(config backend.Config) backend.Config {
	if settings.DeviceName != "" {
		config.DeviceName = settings.DeviceName
	}
	if settings.ReceiveDir != "" {
		config.ReceiveDir = settings.ReceiveDir
	}
	if settings.Port != 0 {
		config.Port = settings.Port
	}
	config.Hidden = settings.Hidden
	return config
}

// Store reads and writes settings in a JSON file.
type Store struct {
	path string
}

// NewStore returns a Store that keeps settings at path.
func NewStore(path string) Store {
	return Store{path: path}
}

// DefaultPath is where the settings file lives for the current user.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(configDir, directoryDir, fileName), nil
}

// Load returns the saved settings, or empty settings if none were saved.
func (store Store) Load() (Settings, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("parse settings %s: %w", store.path, err)
	}
	return settings, nil
}

// Save writes settings, replacing the file atomically so a crash cannot leave
// it half written.
func (store Store) Save(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	if err := os.Rename(temporary, store.path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}
