package transfer

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// newTestServer serves the same protocol as the backend on top of a Receiver.
func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	directory := t.TempDir()
	receiver, err := NewReceiver(directory)
	if err != nil {
		t.Fatalf("NewReceiver: %v", err)
	}
	t.Cleanup(func() { _ = receiver.Close() })

	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload", func(writer http.ResponseWriter, request *http.Request) {
		offset, _ := strconv.ParseInt(request.Header.Get("X-File-Offset"), 10, 64)
		if _, err := receiver.WriteChunk(request.Header.Get("X-File-Name"), offset, request.Body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /status", func(writer http.ResponseWriter, request *http.Request) {
		size, err := receiver.Size(request.URL.Query().Get("filename"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.Header().Set("X-File-Offset", strconv.FormatInt(size, 10))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, directory
}

func writeSourceFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	return path
}

func TestSenderSendsFileInChunks(t *testing.T) {
	server, directory := newTestServer(t)
	content := bytes.Repeat([]byte("deckdrop"), 1000)
	path := writeSourceFile(t, "save.bin", content)

	sender := Sender{ChunkSize: 1024}
	if err := sender.Send(context.Background(), server.Listener.Addr().String(), path); err != nil {
		t.Fatalf("Send: %v", err)
	}

	received, err := os.ReadFile(filepath.Join(directory, "save.bin"))
	if err != nil {
		t.Fatalf("read received file: %v", err)
	}
	if !bytes.Equal(received, content) {
		t.Errorf("received %d bytes, want %d identical bytes", len(received), len(content))
	}
}

func TestSenderSendsEmptyFile(t *testing.T) {
	server, directory := newTestServer(t)
	path := writeSourceFile(t, "empty.txt", nil)

	if err := (Sender{}).Send(context.Background(), server.Listener.Addr().String(), path); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if _, err := os.Stat(filepath.Join(directory, "empty.txt")); err != nil {
		t.Errorf("empty file was not created: %v", err)
	}
}

func TestSenderResumesFromReceivedOffset(t *testing.T) {
	server, directory := newTestServer(t)
	content := bytes.Repeat([]byte("0123456789"), 500)
	path := writeSourceFile(t, "resume.bin", content)
	if err := os.WriteFile(filepath.Join(directory, "resume.bin"), content[:2000], 0o600); err != nil {
		t.Fatalf("seed partial file: %v", err)
	}

	var uploadOffsets []string
	recording := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			uploadOffsets = append(uploadOffsets, request.Header.Get("X-File-Offset"))
		}
		server.Config.Handler.ServeHTTP(writer, request)
	}))
	defer recording.Close()

	sender := Sender{ChunkSize: 1500}
	if err := sender.Send(context.Background(), recording.Listener.Addr().String(), path); err != nil {
		t.Fatalf("Send: %v", err)
	}

	received, err := os.ReadFile(filepath.Join(directory, "resume.bin"))
	if err != nil {
		t.Fatalf("read received file: %v", err)
	}
	if !bytes.Equal(received, content) {
		t.Errorf("received file differs from source")
	}
	if len(uploadOffsets) == 0 || uploadOffsets[0] != "2000" {
		t.Errorf("upload offsets = %v, want first offset 2000", uploadOffsets)
	}
}

func TestSenderReportsServerRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()
	path := writeSourceFile(t, "file.txt", []byte("data"))

	if err := (Sender{}).Send(context.Background(), server.Listener.Addr().String(), path); err == nil {
		t.Fatal("Send succeeded against a failing server")
	}
}

func TestSenderStopsWhenContextCancelled(t *testing.T) {
	server, _ := newTestServer(t)
	path := writeSourceFile(t, "file.txt", []byte("data"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (Sender{}).Send(ctx, server.Listener.Addr().String(), path)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Send error = %v, want context.Canceled", err)
	}
}
