package pairing

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

const requestTimeout = 15 * time.Second

// ErrRejected means the device refused the code.
var ErrRejected = errors.New("the device rejected the pairing code")

// Pair pairs with the device at address ("host:port") using the code it shows.
// self is this device's identity and name is how it should appear on the other
// device. The returned Peer is the other device, with the certificate
// fingerprint to pin.
func Pair(ctx context.Context, self *identity.Identity, name, address, code string) (trust.Peer, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	peerFingerprint, err := learnFingerprint(ctx, self, address)
	if err != nil {
		return trust.Peer{}, err
	}

	body, err := json.Marshal(struct {
		Name  string `json:"name"`
		Proof string `json:"proof"`
	}{name, Proof(code, self.Fingerprint(), peerFingerprint)})
	if err != nil {
		return trust.Peer{}, fmt.Errorf("encode pairing request: %w", err)
	}

	// The connection is pinned to the certificate the proof was made for.
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: identity.ClientConfig(peerFingerprint, self)}}
	defer client.CloseIdleConnections()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+address+"/pair", bytes.NewReader(body))
	if err != nil {
		return trust.Peer{}, fmt.Errorf("build pairing request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return trust.Peer{}, fmt.Errorf("pair with %s: %w", address, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusForbidden {
		return trust.Peer{}, ErrRejected
	}
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 200))
		return trust.Peer{}, fmt.Errorf("pair with %s: %s: %s", address, response.Status, strings.TrimSpace(string(message)))
	}

	var answer struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&answer); err != nil {
		return trust.Peer{}, fmt.Errorf("read pairing response: %w", err)
	}
	return trust.Peer{Fingerprint: peerFingerprint, Name: answer.Name}, nil
}

const maxResponseBytes = 1 << 10

// learnFingerprint connects once to see which certificate the device uses.
func learnFingerprint(ctx context.Context, self *identity.Identity, address string) (string, error) {
	dialer := tls.Dialer{Config: identity.ClientConfig("", self)}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", address, err)
	}
	defer func() { _ = connection.Close() }()

	state := connection.(*tls.Conn).ConnectionState()
	return identity.PeerFingerprint(&state), nil
}
