package backend

import (
	"bytes"
	"context"
	"errors"
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
	if err := sender.Send(context.Background(), testBackend.Addr(), path, nil); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got := receivedFile(t, testBackend, "game.sav"); !bytes.Equal(got, content) {
		t.Errorf("received %d bytes, want %d identical bytes", len(got), len(content))
	}
}

func TestSenderResumesAnInterruptedUpload(t *testing.T) {
	testBackend := startTestBackend(t)
	content := randomBytes(t, 400_000)
	path := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	sender := transfer.Sender{ChunkSize: 64 * 1024}

	ctx, cancel := context.WithCancel(context.Background())
	err := sender.Send(ctx, testBackend.Addr(), path, func(sent, _ int64) {
		if sent > 150_000 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("interrupted Send error = %v, want context.Canceled", err)
	}
	if partial := receivedFile(t, testBackend, "big.bin"); len(partial) >= len(content) {
		t.Fatalf("received %d bytes before resuming, want a partial file", len(partial))
	}

	if err := sender.Send(context.Background(), testBackend.Addr(), path, nil); err != nil {
		t.Fatalf("resumed Send: %v", err)
	}

	if got := receivedFile(t, testBackend, "big.bin"); !bytes.Equal(got, content) {
		t.Errorf("received %d bytes, want %d identical bytes", len(got), len(content))
	}
}
