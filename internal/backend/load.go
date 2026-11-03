package backend

import (
	"errors"
	"fmt"
	"io"

	"github.com/alecthomas/kong"
)

const (
	portEnvironmentVariable       = "DECKDROP_PORT"
	receiveDirEnvironmentVariable = "DECKDROP_RECEIVE_DIR"
	deviceNameEnvironmentVariable = "DECKDROP_DEVICE_NAME"

	maxPort = 65535
)

// ErrHelp is returned by LoadConfig after printing usage because help was
// requested.
var ErrHelp = errors.New("help requested")

// options describes the command line. Kong resolves each value from the flag,
// then the environment variable, then the default.
type options struct {
	Port int    `help:"TCP port to listen on (1-${max_port})." env:"DECKDROP_PORT" default:"${default_port}"`
	Dir  string `help:"Directory received files are saved to." env:"DECKDROP_RECEIVE_DIR" default:"${default_dir}"`
	Name string `help:"Device name shown to other devices." env:"DECKDROP_DEVICE_NAME" default:"${default_name}"`
}

// Validate is called by kong after parsing.
func (parsed *options) Validate() error {
	if parsed.Port < 1 || parsed.Port > maxPort {
		return fmt.Errorf("--port must be between 1 and %d, got %d", maxPort, parsed.Port)
	}
	if parsed.Dir == "" {
		return errors.New("--dir must not be empty")
	}
	if parsed.Name == "" {
		return errors.New("--name must not be empty")
	}
	return nil
}

// helpExit is panicked by kong's exit hook so LoadConfig can return instead of
// the process exiting.
type helpExit struct{}

// LoadConfig builds a Config from args, environment variables and defaults, in
// that order of precedence. Usage is written to output. It returns ErrHelp
// when help was requested.
func LoadConfig(args []string, output io.Writer) (config Config, err error) {
	defaults := DefaultConfig()

	var parsed options
	parser, err := kong.New(&parsed,
		kong.Name("deckdrop"),
		kong.Description("Receive files from other devices on your network."),
		kong.Writers(output, output),
		kong.Vars{
			"default_port": fmt.Sprint(defaults.Port),
			"default_dir":  defaults.ReceiveDir,
			"default_name": defaults.DeviceName,
			"max_port":     fmt.Sprint(maxPort),
		},
		kong.Exit(func(int) { panic(helpExit{}) }),
	)
	if err != nil {
		return Config{}, fmt.Errorf("build command line: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if _, isHelp := recovered.(helpExit); !isHelp {
				panic(recovered)
			}
			config, err = Config{}, ErrHelp
		}
	}()

	if _, err := parser.Parse(args); err != nil {
		return Config{}, err
	}

	defaults.Port = parsed.Port
	defaults.ReceiveDir = parsed.Dir
	defaults.DeviceName = parsed.Name
	return defaults, nil
}
