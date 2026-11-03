package backend

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

const (
	portEnvironmentVariable       = "DECKDROP_PORT"
	receiveDirEnvironmentVariable = "DECKDROP_RECEIVE_DIR"
	deviceNameEnvironmentVariable = "DECKDROP_DEVICE_NAME"

	maxPort = 65535
)

// LoadConfig builds a Config from the defaults, then environment variables,
// then command-line flags, each overriding the previous. Usage and flag errors
// are written to output. It returns flag.ErrHelp when help was requested.
func LoadConfig(args []string, environment func(string) string, output io.Writer) (Config, error) {
	config := DefaultConfig()

	if value := environment(portEnvironmentVariable); value != "" {
		port, err := parsePort(value)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", portEnvironmentVariable, err)
		}
		config.Port = port
	}
	if value := environment(receiveDirEnvironmentVariable); value != "" {
		config.ReceiveDir = value
	}
	if value := environment(deviceNameEnvironmentVariable); value != "" {
		config.DeviceName = value
	}

	flags := flag.NewFlagSet("deckdrop", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Func("port", fmt.Sprintf("TCP port to listen on (env %s, default %d)", portEnvironmentVariable, config.Port),
		func(value string) error {
			port, err := parsePort(value)
			if err != nil {
				return err
			}
			config.Port = port
			return nil
		})
	flags.Func("dir", fmt.Sprintf("directory received files are saved to (env %s, default %q)", receiveDirEnvironmentVariable, config.ReceiveDir),
		func(value string) error {
			if value == "" {
				return errors.New("must not be empty")
			}
			config.ReceiveDir = value
			return nil
		})
	flags.Func("name", fmt.Sprintf("device name shown to other devices (env %s, default %q)", deviceNameEnvironmentVariable, config.DeviceName),
		func(value string) error {
			if value == "" {
				return errors.New("must not be empty")
			}
			config.DeviceName = value
			return nil
		})

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if flags.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	return config, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > maxPort {
		return 0, fmt.Errorf("%q is not a port between 1 and %d", value, maxPort)
	}
	return port, nil
}
