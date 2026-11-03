package backend

import (
	"os"
	"strings"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/offer"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

const (
	defaultPort         = 8080
	defaultReceiveDir   = "received_files"
	defaultMaxChunkSize = 64 << 20 // 64 MiB
	fallbackDeviceName  = "deckdrop"
)

// ReceiptListener is told about each file that has data written to it.
type ReceiptListener interface {
	Received(name string)
}

// Config holds the settings for the backend.
type Config struct {
	// Port is the TCP port the HTTP server listens on. Zero picks a free port.
	Port int

	// ReceiveDir is the directory received files are written to.
	ReceiveDir string

	// MaxChunkSize is the largest accepted upload body in bytes. Zero uses the
	// default.
	MaxChunkSize int64

	// DeviceName is how this device appears to others on the network.
	DeviceName string

	// Headless runs without the desktop window.
	Headless bool

	// Offers decides whether files offered by paired devices are accepted.
	// When set, nothing is written until an offer is accepted. Nil accepts
	// anything from a paired device.
	Offers offer.Decider

	// AcceptPaired makes Serve accept every offer from a paired device
	// instead of declining them, for devices with nobody to ask.
	AcceptPaired bool

	// PairOnStart makes Serve log a one-time pairing code at startup.
	PairOnStart bool

	// Receipts is told when a file is received. Nil disables notifications.
	Receipts ReceiptListener

	// Identity is the certificate the server presents. Nil serves plain HTTP,
	// which is only meant for tests.
	Identity *identity.Identity

	// Trust and Pairing restrict uploads to paired devices and let devices
	// pair. Both must be set; if either is nil uploads are open to everyone,
	// which is only meant for tests.
	Trust   *trust.Store
	Pairing *pairing.Manager

	// Hidden stops the device announcing itself even if there is an
	// Announcer. Paired devices that know its address can still send.
	Hidden bool

	// Announcer advertises the device on the network. Nil disables announcing.
	Announcer Announcer
}

func (config Config) maxChunkSize() int64 {
	if config.MaxChunkSize <= 0 {
		return defaultMaxChunkSize
	}
	return config.MaxChunkSize
}

// DefaultConfig returns the configuration used when nothing is overridden.
// The device name defaults to the machine's hostname.
func DefaultConfig() Config {
	hostname, err := os.Hostname()
	deviceName := shortHostname(hostname)
	if err != nil || deviceName == "" {
		deviceName = fallbackDeviceName
	}
	return Config{Port: defaultPort, ReceiveDir: defaultReceiveDir, MaxChunkSize: defaultMaxChunkSize, DeviceName: deviceName}
}

// shortHostname drops the domain part, e.g. "deck.local" becomes "deck".
func shortHostname(hostname string) string {
	shortName, _, _ := strings.Cut(hostname, ".")
	return shortName
}
