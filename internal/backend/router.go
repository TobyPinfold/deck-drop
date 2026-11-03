package backend

import (
	"log/slog"
	"net/http"

	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

func newRouter(receiver *transfer.Receiver, logger *slog.Logger, config Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("POST /upload", uploadHandler{receiver: receiver, log: logger, directory: config.ReceiveDir, maxChunkSize: config.maxChunkSize(), receipts: config.Receipts})
	mux.Handle("GET /status", statusHandler{receiver: receiver, log: logger})
	return mux
}

func handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
