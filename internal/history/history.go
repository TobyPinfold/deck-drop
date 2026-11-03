// Package history remembers which files DeckDrop has received, so the app can
// list them without listing everything else in the receive folder.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	directoryName = "DeckDrop"
	fileName      = "received.json"
)

// Store keeps the names of received files in a JSON file.
type Store struct {
	path string
}

// NewStore returns a Store that keeps names at path.
func NewStore(path string) Store {
	return Store{path: path}
}

// DefaultPath is where the history lives for the current user.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(configDir, directoryName, fileName), nil
}

// Load returns the saved names, or none if nothing was saved.
func (store Store) Load() ([]string, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read received history: %w", err)
	}

	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, fmt.Errorf("parse received history %s: %w", store.path, err)
	}
	return names, nil
}

// Save replaces the saved names, writing atomically.
func (store Store) Save(names []string) error {
	data, err := json.Marshal(names)
	if err != nil {
		return fmt.Errorf("encode received history: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write received history: %w", err)
	}
	if err := os.Rename(temporary, store.path); err != nil {
		return fmt.Errorf("replace received history: %w", err)
	}
	return nil
}
