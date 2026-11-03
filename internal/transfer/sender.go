package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/TobyPinfold/deck-drop/internal/identity"
)

const defaultChunkSize = 8 << 20

// ErrNotPaired means the device refused the upload because this device has
// not been paired with it.
var ErrNotPaired = errors.New("this device is not paired with the receiver")

// Sender uploads files to another DeckDrop device in chunks.
type Sender struct {
	// Client performs the requests. It overrides Secure.
	Client *http.Client

	// Secure connects over HTTPS. Without it (and without Client) the sender
	// uses plain HTTP, which is only meant for tests.
	Secure bool

	// Identity is the client certificate presented to devices, so they can
	// tell who is sending.
	Identity *identity.Identity

	// ChunkSize is the largest number of bytes sent per request. It must not
	// exceed the receiver's chunk limit. It defaults to 8 MiB.
	ChunkSize int64
}

// Send uploads the file at path to the device listening on address
// ("host:port"). With Secure, the device must present the certificate with
// fingerprint; an empty fingerprint accepts any certificate. If the device already has part of the file, the upload
// resumes where it left off. Cancelling ctx stops the upload; sending the same
// file again continues from what the device received.
//
// progress, if not nil, is called with the bytes the device has and the file's
// total size: first with the resume offset, then as data is sent.
func (sender Sender) Send(ctx context.Context, address, fingerprint, path string, progress func(sent, total int64)) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	name := filepath.Base(path)
	size := info.Size()
	baseURL := sender.scheme() + "://" + address
	client := sender.clientFor(fingerprint)
	defer client.CloseIdleConnections()

	offset, err := sender.receivedSize(ctx, client, baseURL, name)
	if err != nil {
		return err
	}
	if offset > size {
		return fmt.Errorf("device already has %d bytes of %q, more than the %d to send", offset, name, size)
	}

	if progress == nil {
		progress = func(int64, int64) {}
	}
	progress(offset, size)

	chunkSize := sender.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	// An empty file still needs one request so the device creates it.
	for sent := false; !sent || offset < size; sent = true {
		length := min(chunkSize, size-offset)
		chunk := &progressReader{
			reader:   io.NewSectionReader(file, offset, length),
			sent:     offset,
			total:    size,
			progress: progress,
		}
		if err := sender.uploadChunk(ctx, client, baseURL, name, offset, length, chunk); err != nil {
			return err
		}
		offset += length
	}
	return nil
}

func (sender Sender) receivedSize(ctx context.Context, client *http.Client, baseURL, name string) (int64, error) {
	query := "?filename=" + url.QueryEscape(name)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/status"+query, nil)
	if err != nil {
		return 0, fmt.Errorf("build status request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("check progress of %q: %w", name, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusForbidden {
		return 0, ErrNotPaired
	}
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("check progress of %q: %s", name, response.Status)
	}
	offset, err := strconv.ParseInt(response.Header.Get("X-File-Offset"), 10, 64)
	if err != nil {
		return 0, errors.New("device returned an invalid file offset")
	}
	return offset, nil
}

func (sender Sender) uploadChunk(ctx context.Context, client *http.Client, baseURL, name string, offset, length int64, chunk io.Reader) error {
	if length == 0 {
		chunk = http.NoBody
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/upload", chunk)
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}
	request.ContentLength = length
	request.Header.Set("X-File-Name", name)
	request.Header.Set("X-File-Offset", strconv.FormatInt(offset, 10))

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("upload %q at %d: %w", name, offset, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusForbidden {
		return ErrNotPaired
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("upload %q at %d: %s", name, offset, response.Status)
	}
	return nil
}

func (sender Sender) scheme() string {
	if sender.Secure {
		return "https"
	}
	return "http"
}

func (sender Sender) clientFor(fingerprint string) *http.Client {
	if sender.Client != nil {
		return sender.Client
	}
	if !sender.Secure {
		return http.DefaultClient
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: identity.ClientConfig(fingerprint, sender.Identity)}}
}

// progressReader reports how much of the file has been read as it is uploaded.
type progressReader struct {
	reader   io.Reader
	sent     int64
	total    int64
	progress func(sent, total int64)
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	count, err := reader.reader.Read(buffer)
	if count > 0 {
		reader.sent += int64(count)
		reader.progress(reader.sent, reader.total)
	}
	return count, err
}
