// Package httpapi provides the backend HTTP routes and handlers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

// ReceiptListener is told about each file that has data written to it.
type ReceiptListener interface {
	Received(name string)
}

// NewHandler creates the backend HTTP routes for receiver.
func NewHandler(
	receiver *transfer.Receiver,
	logger *slog.Logger,
	directory string,
	maxChunkSize int64,
	receipts ReceiptListener,
	access *Access,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	if access != nil {
		mux.HandleFunc("POST /pair", access.handlePair)
	}
	mux.Handle("POST /upload", access.require(uploadHandler{
		receiver:     receiver,
		log:          logger,
		directory:    directory,
		maxChunkSize: maxChunkSize,
		receipts:     receipts,
	}))
	mux.Handle("GET /status", access.require(statusHandler{receiver: receiver, log: logger}))
	return mux
}

func handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
