// Package pairing implements pairing two devices with a one-time code.
//
// The device being paired with shows a six digit code. The other device proves
// it knows the code by sending an HMAC over both certificate fingerprints, so
// the code itself never crosses the network and a device in the middle, which
// would have a different certificate, cannot reuse the proof.
package pairing

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

const (
	codeLifetime = 2 * time.Minute
	maxAttempts  = 5
	codeDigits   = 6
)

// ErrNoCode means no code is active: none was started, it expired, was
// already used, or too many wrong proofs were sent.
var ErrNoCode = errors.New("no pairing code is active")

// ErrWrongCode means the proof did not match the active code.
var ErrWrongCode = errors.New("wrong pairing code")

// Code is a one-time pairing code.
type Code struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Manager issues one-time codes and checks proofs against them.
type Manager struct {
	fingerprint string
	now         func() time.Time

	mu       sync.Mutex
	code     Code
	attempts int
}

// NewManager returns a Manager for the device with the given certificate
// fingerprint.
func NewManager(fingerprint string) *Manager {
	return &Manager{fingerprint: fingerprint, now: time.Now}
}

// Fingerprint is this device's certificate fingerprint.
func (manager *Manager) Fingerprint() string { return manager.fingerprint }

// NewCode starts pairing, replacing any active code.
func (manager *Manager) NewCode() (Code, error) {
	number, err := rand.Int(rand.Reader, new(big.Int).Exp(big.NewInt(10), big.NewInt(codeDigits), nil))
	if err != nil {
		return Code{}, fmt.Errorf("generate pairing code: %w", err)
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.code = Code{
		Value:     fmt.Sprintf("%0*d", codeDigits, number),
		ExpiresAt: manager.now().Add(codeLifetime),
	}
	manager.attempts = 0
	return manager.code, nil
}

// Cancel ends pairing, discarding any active code.
func (manager *Manager) Cancel() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.code = Code{}
}

// Verify checks proof from the device with clientFingerprint. A correct proof
// uses up the code. A wrong one counts as an attempt, and the code is
// discarded after too many so it cannot be guessed.
func (manager *Manager) Verify(clientFingerprint, proof string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.code.Value == "" || !manager.now().Before(manager.code.ExpiresAt) {
		manager.code = Code{}
		return ErrNoCode
	}

	expected := Proof(manager.code.Value, clientFingerprint, manager.fingerprint)
	if hmac.Equal([]byte(expected), []byte(proof)) {
		manager.code = Code{}
		return nil
	}

	manager.attempts++
	if manager.attempts >= maxAttempts {
		manager.code = Code{}
	}
	return ErrWrongCode
}

// Proof is what a device sends to show it knows code, bound to the
// fingerprints of both devices.
func Proof(code, clientFingerprint, serverFingerprint string) string {
	mac := hmac.New(sha256.New, []byte(code))
	mac.Write([]byte(clientFingerprint + "\n" + serverFingerprint))
	return hex.EncodeToString(mac.Sum(nil))
}
