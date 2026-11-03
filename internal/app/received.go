package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"
)

// ReceivedFile is a file in the receive directory.
type ReceivedFile struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// ReceiveDir is the directory received files are saved to.
func (app *App) ReceiveDir() string {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.receiveDir
}

// Received records that a file was received. It implements
// backend.ReceiptListener.
func (app *App) Received(name string) {
	app.mu.Lock()
	if slices.Contains(app.received, name) {
		app.mu.Unlock()
		return
	}
	app.received = append(app.received, name)
	names := slices.Clone(app.received)
	app.mu.Unlock()

	if app.options.HistoryStore == nil {
		return
	}
	if err := app.options.HistoryStore.Save(names); err != nil {
		app.log.Warn("failed to save received file history", "error", err)
	}
}

// ReceivedFiles lists the files DeckDrop has received that are still in the
// receive directory, newest first.
func (app *App) ReceivedFiles() ([]ReceivedFile, error) {
	entries, err := os.ReadDir(app.ReceiveDir())
	if errors.Is(err, fs.ErrNotExist) {
		return []ReceivedFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read receive directory: %w", err)
	}

	app.mu.Lock()
	received := slices.Clone(app.received)
	app.mu.Unlock()

	files := make([]ReceivedFile, 0, len(entries))
	for _, entry := range entries {
		if !slices.Contains(received, entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		files = append(files, ReceivedFile{Name: info.Name(), Size: info.Size(), Modified: info.ModTime()})
	}

	slices.SortFunc(files, func(first, second ReceivedFile) int {
		if byTime := second.Modified.Compare(first.Modified); byTime != 0 {
			return byTime
		}
		return strings.Compare(first.Name, second.Name)
	})
	return files, nil
}
