package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

const (
	maxPairRequestBytes = 1 << 10
	maxPeerNameLength   = 64
)

// PairRequest is what a device sends to pair.
type PairRequest struct {
	Name  string `json:"name"`
	Proof string `json:"proof"`
}

// PairResponse is the paired device's answer.
type PairResponse struct {
	Name string `json:"name"`
}

// Paired lists the devices allowed to send files.
type Paired interface {
	Has(fingerprint string) bool
	Add(peer trust.Peer) error
}

// Pairer checks pairing proofs.
type Pairer interface {
	Verify(clientFingerprint, proof string) error
}

// Access controls who may use the upload routes and lets devices pair.
type Access struct {
	Paired     Paired
	Pairer     Pairer
	DeviceName string
}

// require lets a request through only from a paired device, which is
// recognised by the certificate it presented in the TLS handshake.
func (access *Access) require(next http.Handler) http.Handler {
	if access == nil {
		return next
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !access.Paired.Has(identity.PeerFingerprint(request.TLS)) {
			http.Error(writer, "this device has not paired with the receiver", http.StatusForbidden)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (access *Access) handlePair(writer http.ResponseWriter, request *http.Request) {
	fingerprint := identity.PeerFingerprint(request.TLS)
	if fingerprint == "" {
		http.Error(writer, "a client certificate is required to pair", http.StatusUnauthorized)
		return
	}

	var body PairRequest
	request.Body = http.MaxBytesReader(writer, request.Body, maxPairRequestBytes)
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "invalid pairing request", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > maxPeerNameLength {
		http.Error(writer, "invalid device name", http.StatusBadRequest)
		return
	}

	if err := access.Pairer.Verify(fingerprint, body.Proof); err != nil {
		if errors.Is(err, pairing.ErrNoCode) || errors.Is(err, pairing.ErrWrongCode) {
			http.Error(writer, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(writer, "pairing failed", http.StatusInternalServerError)
		return
	}
	if err := access.Paired.Add(trust.Peer{Fingerprint: fingerprint, Name: name}); err != nil {
		http.Error(writer, "could not save the pairing", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(PairResponse{Name: access.DeviceName})
}
