package backend

import "os"

const (
	defaultPort        = 8080
	fallbackDeviceName = "deckdrop"
)

// Config holds the settings for the backend.
type Config struct {
	// Port is the TCP port the HTTP server listens on. Zero picks a free port.
	Port int

	// DeviceName is how this device appears to others on the network.
	DeviceName string

	// Announcer advertises the device on the network. Nil disables announcing.
	Announcer Announcer
}

// DefaultConfig returns the configuration used when nothing is overridden.
// The device name defaults to the machine's hostname.
func DefaultConfig() Config {
	deviceName, err := os.Hostname()
	if err != nil || deviceName == "" {
		deviceName = fallbackDeviceName
	}
	return Config{Port: defaultPort, DeviceName: deviceName}
}
