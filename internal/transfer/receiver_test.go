package transfer

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestReceiver(t *testing.T) (*Receiver, string) {
	t.Helper()

	directory := t.TempDir()
	receiver, err := NewReceiver(directory)
	if err != nil {
		t.Fatalf("NewReceiver: %v", err)
	}
	t.Cleanup(func() { _ = receiver.Close() })
	return receiver, directory
}

func randomContent(t *testing.T, size int) []byte {
	t.Helper()

	content := make([]byte, size)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return content
}

func writeChunk(t *testing.T, receiver *Receiver, name string, offset int, chunk []byte) {
	t.Helper()

	written, err := receiver.WriteChunk(name, int64(offset), bytes.NewReader(chunk))
	if err != nil {
		t.Fatalf("WriteChunk(%q, offset %d): %v", name, offset, err)
	}
	if written != int64(len(chunk)) {
		t.Fatalf("WriteChunk wrote %d bytes, want %d", written, len(chunk))
	}
}

func assertFileContent(t *testing.T, directory, name string, want []byte) {
	t.Helper()

	got, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s content differs: got %d bytes, want %d bytes", name, len(got), len(want))
	}
}

func splitIntoChunks(content []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for start := 0; start < len(content); start += chunkSize {
		end := min(start+chunkSize, len(content))
		chunks = append(chunks, content[start:end])
	}
	return chunks
}

func TestNewReceiverCreatesDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "nested", "received")

	receiver, err := NewReceiver(directory)
	if err != nil {
		t.Fatalf("NewReceiver: %v", err)
	}
	t.Cleanup(func() { _ = receiver.Close() })

	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		t.Fatalf("expected %s to be a directory (err: %v)", directory, err)
	}
}

func TestWriteChunksInOrderProducesIdenticalFile(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	content := randomContent(t, 1000)

	offset := 0
	for _, chunk := range splitIntoChunks(content, 300) {
		writeChunk(t, receiver, "save.tar.gz", offset, chunk)
		offset += len(chunk)
	}

	assertFileContent(t, directory, "save.tar.gz", content)
}

func TestWriteChunksOutOfOrderProducesIdenticalFile(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	content := randomContent(t, 900)

	writeChunk(t, receiver, "game.bin", 600, content[600:])
	writeChunk(t, receiver, "game.bin", 0, content[:300])
	writeChunk(t, receiver, "game.bin", 300, content[300:600])

	assertFileContent(t, directory, "game.bin", content)
}

func TestWriteChunkOverwritesExistingBytesAtOffset(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	content := randomContent(t, 600)

	writeChunk(t, receiver, "retry.bin", 0, content)
	writeChunk(t, receiver, "retry.bin", 200, content[200:400])

	assertFileContent(t, directory, "retry.bin", content)
}

func TestInterleavedChunksOfDifferentFilesDoNotInterfere(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	firstContent := randomContent(t, 600)
	secondContent := randomContent(t, 600)

	writeChunk(t, receiver, "first.bin", 0, firstContent[:300])
	writeChunk(t, receiver, "second.bin", 0, secondContent[:300])
	writeChunk(t, receiver, "second.bin", 300, secondContent[300:])
	writeChunk(t, receiver, "first.bin", 300, firstContent[300:])

	assertFileContent(t, directory, "first.bin", firstContent)
	assertFileContent(t, directory, "second.bin", secondContent)
}

func TestWriteChunkRejectsNamesOutsideTheDirectory(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	names := []string{"", ".", "..", "../escape.bin", "nested/file.bin", "/etc/passwd"}

	for _, name := range names {
		_, err := receiver.WriteChunk(name, 0, bytes.NewReader([]byte("data")))

		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("WriteChunk(%q) error = %v, want ErrInvalidName", name, err)
		}
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("directory should stay empty, got %v (err: %v)", entries, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "..", "escape.bin")); err == nil {
		t.Fatal("file escaped the receive directory")
	}
}

func TestWriteChunkRejectsNegativeOffset(t *testing.T) {
	receiver, _ := newTestReceiver(t)

	_, err := receiver.WriteChunk("file.bin", -1, bytes.NewReader([]byte("data")))

	if !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("error = %v, want ErrInvalidOffset", err)
	}
}

func TestSizeOfUnknownFileIsZero(t *testing.T) {
	receiver, _ := newTestReceiver(t)

	size, err := receiver.Size("never-sent.bin")

	if err != nil || size != 0 {
		t.Fatalf("Size = %d, %v; want 0, nil", size, err)
	}
}

func TestSizeReportsBytesReceived(t *testing.T) {
	receiver, _ := newTestReceiver(t)
	content := randomContent(t, 700)
	writeChunk(t, receiver, "partial.bin", 0, content[:300])

	size, err := receiver.Size("partial.bin")

	if err != nil || size != 300 {
		t.Fatalf("Size = %d, %v; want 300, nil", size, err)
	}
}

func TestSizeRejectsInvalidNames(t *testing.T) {
	receiver, _ := newTestReceiver(t)

	_, err := receiver.Size("../secret.bin")

	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}

func TestResumingFromReportedSizeCompletesFile(t *testing.T) {
	receiver, directory := newTestReceiver(t)
	content := randomContent(t, 1000)
	writeChunk(t, receiver, "resumed.bin", 0, content[:450])

	resumeOffset, err := receiver.Size("resumed.bin")
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	writeChunk(t, receiver, "resumed.bin", int(resumeOffset), content[resumeOffset:])

	assertFileContent(t, directory, "resumed.bin", content)
}
