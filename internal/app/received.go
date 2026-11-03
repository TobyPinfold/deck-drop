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

// ReceivedFiles lists the files in the receive directory, newest first.
func (app *App) ReceivedFiles() ([]ReceivedFile, error) {
	entries, err := os.ReadDir(app.ReceiveDir())
	if errors.Is(err, fs.ErrNotExist) {
		return []ReceivedFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read receive directory: %w", err)
	}

	files := make([]ReceivedFile, 0, len(entries))
	for _, entry := range entries {
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
