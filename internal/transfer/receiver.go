// Package transfer receives files sent in chunks.
package transfer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
)

const (
	directoryPermissions = 0o755
	filePermissions      = 0o644
)

var (
	// ErrInvalidName is returned for file names that are not a plain file name.
	ErrInvalidName = errors.New("invalid file name")

	// ErrInvalidOffset is returned for negative chunk offsets.
	ErrInvalidOffset = errors.New("invalid chunk offset")
)

// Receiver writes incoming file chunks into a single directory.
type Receiver struct {
	root *os.Root

	// openMutex serializes opening files: concurrent creation of the same
	// new file through os.Root can fail spuriously with ENOENT.
	openMutex sync.Mutex
}

// NewReceiver creates directory if needed and receives files into it.
func NewReceiver(directory string) (*Receiver, error) {
	if err := os.MkdirAll(directory, directoryPermissions); err != nil {
		return nil, fmt.Errorf("create receive directory: %w", err)
	}

	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open receive directory: %w", err)
	}
	return &Receiver{root: root}, nil
}

// WriteChunk writes chunk into the named file starting at offset, creating
// the file if needed. It returns the number of bytes written.
func (receiver *Receiver) WriteChunk(name string, offset int64, chunk io.Reader) (int64, error) {
	if !isPlainFileName(name) {
		return 0, ErrInvalidName
	}
	if offset < 0 {
		return 0, ErrInvalidOffset
	}

	receiver.openMutex.Lock()
	file, err := receiver.root.OpenFile(name, os.O_CREATE|os.O_WRONLY, filePermissions)
	receiver.openMutex.Unlock()
	if err != nil {
		return 0, fmt.Errorf("open %q: %w", name, err)
	}

	written, copyErr := io.Copy(io.NewOffsetWriter(file, offset), chunk)
	closeErr := file.Close()
	if copyErr != nil {
		return written, fmt.Errorf("write %q: %w", name, copyErr)
	}
	if closeErr != nil {
		return written, fmt.Errorf("close %q: %w", name, closeErr)
	}
	return written, nil
}

// Size returns how many bytes of the named file have been received, or zero
// if nothing has been received yet. Chunks are expected in order, so the size
// is the offset a sender should resume from.
func (receiver *Receiver) Size(name string) (int64, error) {
	if !isPlainFileName(name) {
		return 0, ErrInvalidName
	}

	info, err := receiver.root.Stat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("stat %q: %w", name, err)
	}
	return info.Size(), nil
}

// Close releases the receive directory.
func (receiver *Receiver) Close() error {
	return receiver.root.Close()
}
