package backend

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServeStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() { done <- Serve(ctx, discardLogger(), Config{Port: 0, ReceiveDir: t.TempDir()}) }()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the context was cancelled")
	}
}

func TestServeReportsStartFailure(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port

	err = Serve(context.Background(), discardLogger(), Config{Port: port, ReceiveDir: t.TempDir()})

	if err == nil {
		t.Fatal("Serve succeeded although the port was taken")
	}
}
