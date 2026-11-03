package backend

const defaultPort = 8080

// Config holds the settings for the backend.
type Config struct {
	// Port is the TCP port the HTTP server listens on. Zero picks a free port.
	Port int
}

// DefaultConfig returns the configuration used when nothing is overridden.
func DefaultConfig() Config {
	return Config{Port: defaultPort}
}
