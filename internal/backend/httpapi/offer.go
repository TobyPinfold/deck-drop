package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/offer"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

const (
	maxOfferRequestBytes = 256 << 10
	maxOfferFiles        = 1000

	// decisionTimeout bounds how long an offer waits for the user.
	decisionTimeout = 2 * time.Minute
)

// OfferRequest is a device asking to send files.
type OfferRequest struct {
	Files []offer.File `json:"files"`
}

// handleOffer asks the user whether to accept the files and, if so, allows
// the device to upload them. It answers once the decision is made, so the
// sender simply waits.
func (access *Access) handleOffer(writer http.ResponseWriter, request *http.Request) {
	fingerprint := identity.PeerFingerprint(request.TLS)
	peer, _ := access.Paired.Get(fingerprint)

	var body OfferRequest
	request.Body = http.MaxBytesReader(writer, request.Body, maxOfferRequestBytes)
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil || len(body.Files) == 0 || len(body.Files) > maxOfferFiles {
		http.Error(writer, "invalid offer", http.StatusBadRequest)
		return
	}
	for _, file := range body.Files {
		if !transfer.ValidName(file.Name) || file.Size < 0 {
			http.Error(writer, "invalid file in offer: "+file.Name, http.StatusBadRequest)
			return
		}
	}

	// Asking again about files that were already accepted, such as after a
	// reconnect, does not prompt again.
	if !access.Grants.Covers(fingerprint, body.Files) {
		ctx, cancel := context.WithTimeout(request.Context(), decisionTimeout)
		defer cancel()
		if !access.Offers.Decide(ctx, offer.Offer{From: peer, Files: body.Files}) {
			http.Error(writer, "the receiver declined the files", http.StatusForbidden)
			return
		}
		access.Grants.Allow(fingerprint, body.Files)
	}
	writer.WriteHeader(http.StatusNoContent)
}

// requireOffer only lets a request through for a file the device offered and
// the receiver accepted. nameOf picks the file name out of the request.
func (access *Access) requireOffer(nameOf func(*http.Request) string, next http.Handler) http.Handler {
	if access == nil || access.Offers == nil {
		return next
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fingerprint := identity.PeerFingerprint(request.TLS)
		name := nameOf(request)
		size, granted := access.Grants.Size(fingerprint, name)
		if !granted {
			http.Error(writer, "the receiver has not accepted this file", http.StatusPreconditionRequired)
			return
		}

		if request.Method != http.MethodPost {
			next.ServeHTTP(writer, request)
			return
		}

		// A chunk must stay within the size that was accepted.
		offset, err := strconv.ParseInt(request.Header.Get(fileOffsetHeader), 10, 64)
		if err != nil || request.ContentLength < 0 {
			http.Error(writer, "missing or invalid offset or content length", http.StatusBadRequest)
			return
		}
		end := offset + request.ContentLength
		if offset < 0 || end > size {
			http.Error(writer, "upload is larger than the accepted file", http.StatusBadRequest)
			return
		}

		recorder := &statusRecorder{ResponseWriter: writer}
		next.ServeHTTP(recorder, request)
		if recorder.status == http.StatusNoContent && end >= size {
			access.Grants.Complete(fingerprint, name)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func uploadName(request *http.Request) string { return request.Header.Get(fileNameHeader) }

func statusName(request *http.Request) string { return request.URL.Query().Get("filename") }
