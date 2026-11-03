// Package trust remembers the devices the user has paired with.
package trust

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

const (
	directoryName = "DeckDrop"
	fileName      = "paired.json"
)

// Peer is a paired device, identified by its certificate fingerprint.
type Peer struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`

	// AutoAccept accepts files from this device without asking.
	AutoAccept bool `json:"autoAccept"`
}

// Store is the set of paired devices, saved to a JSON file.
type Store struct {
	path string

	mu       sync.Mutex
	peers    []Peer
	onChange func([]Peer)
}

// DefaultPath is where the paired devices live for the current user.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(configDir, directoryName, fileName), nil
}

// Open loads the store saved at path. A missing file is an empty store; an
// empty path gives a store that is not saved.
func Open(path string) (*Store, error) {
	store := &Store{path: path}
	if path == "" {
		return store, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read paired devices: %w", err)
	}
	if err := json.Unmarshal(data, &store.peers); err != nil {
		return nil, fmt.Errorf("parse paired devices %s: %w", path, err)
	}
	return store, nil
}

// OnChange sets a function called, without the store locked, after the
// paired devices change.
func (store *Store) OnChange(notify func([]Peer)) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.onChange = notify
}

// Has reports whether the device with fingerprint is paired.
func (store *Store) Has(fingerprint string) bool {
	if fingerprint == "" {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return slices.ContainsFunc(store.peers, func(peer Peer) bool { return peer.Fingerprint == fingerprint })
}

// Get returns the paired device with fingerprint.
func (store *Store) Get(fingerprint string) (Peer, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	index := slices.IndexFunc(store.peers, func(peer Peer) bool { return peer.Fingerprint == fingerprint })
	if fingerprint == "" || index < 0 {
		return Peer{}, false
	}
	return store.peers[index], true
}

// SetAutoAccept sets whether files from a paired device are accepted without
// asking. Unknown devices are ignored.
func (store *Store) SetAutoAccept(fingerprint string, autoAccept bool) error {
	return store.change(func(peers []Peer) []Peer {
		for index := range peers {
			if peers[index].Fingerprint == fingerprint {
				peers[index].AutoAccept = autoAccept
			}
		}
		return peers
	})
}

// List returns the paired devices.
func (store *Store) List() []Peer {
	store.mu.Lock()
	defer store.mu.Unlock()
	return slices.Clone(store.peers)
}

// Add pairs a device, replacing any earlier entry with the same fingerprint.
func (store *Store) Add(peer Peer) error {
	return store.change(func(peers []Peer) []Peer {
		peers = slices.DeleteFunc(peers, func(existing Peer) bool {
			if existing.Fingerprint != peer.Fingerprint {
				return false
			}
			peer.AutoAccept = existing.AutoAccept
			return true
		})
		return append(peers, peer)
	})
}

// Remove unpairs the device with fingerprint. Unknown devices are ignored.
func (store *Store) Remove(fingerprint string) error {
	return store.change(func(peers []Peer) []Peer {
		return slices.DeleteFunc(peers, func(existing Peer) bool { return existing.Fingerprint == fingerprint })
	})
}

// change applies edit and saves; if saving fails nothing changes.
func (store *Store) change(edit func([]Peer) []Peer) error {
	store.mu.Lock()
	updated := edit(slices.Clone(store.peers))
	if err := store.save(updated); err != nil {
		store.mu.Unlock()
		return err
	}
	store.peers = updated
	notify := store.onChange
	snapshot := slices.Clone(updated)
	store.mu.Unlock()

	if notify != nil {
		notify(snapshot)
	}
	return nil
}

func (store *Store) save(peers []Peer) error {
	if store.path == "" {
		return nil
	}
	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("encode paired devices: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return fmt.Errorf("create paired devices directory: %w", err)
	}
	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write paired devices: %w", err)
	}
	if err := os.Rename(temporary, store.path); err != nil {
		return fmt.Errorf("replace paired devices: %w", err)
	}
	return nil
}
