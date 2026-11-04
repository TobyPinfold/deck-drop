package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

const (
	fileNameHeader   = "X-File-Name"
	fileOffsetHeader = "X-File-Offset"
)

type uploadHandler struct {
	receiver *transfer.Receiver
	log      *slog.Logger

	// directory is only used to tell the user where files are saved.
	directory string

	maxChunkSize int64

	receipts ReceiptListener
}

// ServeHTTP writes the request body into the named file at the given offset.
func (handler uploadHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.Header.Get(fileNameHeader)
	offset, err := strconv.ParseInt(request.Header.Get(fileOffsetHeader), 10, 64)
	if name == "" || err != nil {
		http.Error(writer, "missing or invalid "+fileNameHeader+" or "+fileOffsetHeader+" header", http.StatusBadRequest)
		return
	}

	if request.ContentLength > handler.maxChunkSize {
		http.Error(writer, "chunk too large", http.StatusRequestEntityTooLarge)
		return
	}
	if offset == 0 {
		handler.log.Info("receiving file", "file", name, "path", filepath.Join(handler.directory, name))
	}

	var body io.Reader = http.MaxBytesReader(writer, request.Body, handler.maxChunkSize)
	progress, tracked := handler.receipts.(ProgressListener)
	fingerprint := identity.PeerFingerprint(request.TLS)
	total, _ := request.Context().Value(acceptedSizeKey{}).(int64)
	if tracked {
		body = &progressReader{reader: body, offset: offset, report: func(received int64) {
			progress.Receiving(fingerprint, name, received, total)
		}}
	}

	if _, err := handler.receiver.WriteChunk(name, offset, body); err != nil {
		if tracked {
			progress.ReceiveStopped(fingerprint, name)
		}
		handler.writeError(writer, name, err)
		return
	}
	if handler.receipts != nil {
		handler.receipts.Received(name)
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler uploadHandler) writeError(writer http.ResponseWriter, name string, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		http.Error(writer, "chunk too large", http.StatusRequestEntityTooLarge)
		return
	}

	if errors.Is(err, transfer.ErrInvalidName) || errors.Is(err, transfer.ErrInvalidOffset) {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	handler.log.Error("failed to write chunk", "file", name, "error", err)
	http.Error(writer, "failed to write chunk", http.StatusInternalServerError)
}

// progressReader reports how far into the file a chunk has been read.
type progressReader struct {
	reader io.Reader
	offset int64
	report func(received int64)
}

func (progress *progressReader) Read(buffer []byte) (int, error) {
	count, err := progress.reader.Read(buffer)
	if count > 0 {
		progress.offset += int64(count)
		progress.report(progress.offset)
	}
	return count, err
}
