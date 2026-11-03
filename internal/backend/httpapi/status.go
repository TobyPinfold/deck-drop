package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

type statusHandler struct {
	receiver *transfer.Receiver
	log      *slog.Logger
}

// ServeHTTP reports in a header how many bytes of a file have been received,
// which is the offset an interrupted upload should resume from.
func (handler statusHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.URL.Query().Get("filename")
	if name == "" {
		http.Error(writer, "missing filename parameter", http.StatusBadRequest)
		return
	}

	size, err := handler.receiver.Size(name)
	if errors.Is(err, transfer.ErrInvalidName) {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		handler.log.Error("failed to read file status", "file", name, "error", err)
		http.Error(writer, "failed to read file status", http.StatusInternalServerError)
		return
	}

	writer.Header().Set(fileOffsetHeader, strconv.FormatInt(size, 10))
	writer.WriteHeader(http.StatusOK)
}
