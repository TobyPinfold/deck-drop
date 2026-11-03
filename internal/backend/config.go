package backend

import (
	"os"
	"strings"
)

const (
	defaultPort         = 8080
	defaultReceiveDir   = "received_files"
	defaultMaxChunkSize = 64 << 20 // 64 MiB
	fallbackDeviceName  = "deckdrop"
)

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
