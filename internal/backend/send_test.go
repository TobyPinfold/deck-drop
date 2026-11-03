package backend

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

func TestSenderDeliversFileToBackend(t *testing.T) {
	testBackend := startTestBackend(t)
	content := randomBytes(t, 300_000)
	path := filepath.Join(t.TempDir(), "game.sav")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	sender := transfer.Sender{ChunkSize: 64 * 1024}
	if err := sender.Send(context.Background(), testBackend.Addr(), path); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got := receivedFile(t, testBackend, "game.sav"); !bytes.Equal(got, content) {
		t.Errorf("received %d bytes, want %d identical bytes", len(got), len(content))
	}
}
