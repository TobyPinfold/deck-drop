package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/backend"
)

func newAppReceivingInto(t *testing.T, directory string) *App {
	t.Helper()

	config := backend.DefaultConfig()
	config.ReceiveDir = directory
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), config, Options{})
}

func writeReceived(t *testing.T, directory, name, content string, modified time.Time) {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatalf("set time of %s: %v", name, err)
	}
}

func TestReceivedFilesListsNewestFirstWithSizes(t *testing.T) {
	directory := t.TempDir()
	now := time.Now()
	writeReceived(t, directory, "old.txt", "12345", now.Add(-time.Hour))
	writeReceived(t, directory, "new.txt", "123", now)
	if err := os.Mkdir(filepath.Join(directory, "subdirectory"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	files, err := newAppReceivingInto(t, directory).ReceivedFiles()

	if err != nil {
		t.Fatalf("ReceivedFiles: %v", err)
	}
	if len(files) != 2 || files[0].Name != "new.txt" || files[0].Size != 3 || files[1].Name != "old.txt" || files[1].Size != 5 {
		t.Errorf("files = %+v, want new.txt (3 bytes) then old.txt (5 bytes), no directories", files)
	}
}

func TestReceivedFilesIsEmptyBeforeAnythingIsReceived(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-created-yet")

	files, err := newAppReceivingInto(t, missing).ReceivedFiles()

	if err != nil || len(files) != 0 {
		t.Errorf("ReceivedFiles = %v, %v; want an empty list and no error", files, err)
	}
}

func TestReceiveDirReportsTheConfiguredDirectory(t *testing.T) {
	directory := t.TempDir()

	if got := newAppReceivingInto(t, directory).ReceiveDir(); got != directory {
		t.Errorf("ReceiveDir = %q, want %q", got, directory)
	}
}
