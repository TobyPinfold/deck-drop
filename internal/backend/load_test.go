package backend

import (
	"bytes"
	"errors"
	"flag"
	"testing"
)

func environmentOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func loadConfig(t *testing.T, args []string, environment map[string]string) (Config, error) {
	t.Helper()

	return LoadConfig(args, environmentOf(environment), &bytes.Buffer{})
}

func TestLoadConfigDefaultsWorkWithNoConfiguration(t *testing.T) {
	config, err := loadConfig(t, nil, nil)

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config != DefaultConfig() {
		t.Fatalf("config = %+v, want defaults %+v", config, DefaultConfig())
	}
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	environment := map[string]string{
		"DECKDROP_PORT":        "9000",
		"DECKDROP_RECEIVE_DIR": "/data/incoming",
		"DECKDROP_DEVICE_NAME": "steam-deck",
	}

	config, err := loadConfig(t, nil, environment)

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.Port != 9000 || config.ReceiveDir != "/data/incoming" || config.DeviceName != "steam-deck" {
		t.Fatalf("config = %+v", config)
	}
}

func TestLoadConfigReadsFlags(t *testing.T) {
	args := []string{"-port", "9100", "-dir", "/tmp/files", "-name", "living-room-pc"}

	config, err := loadConfig(t, args, nil)

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.Port != 9100 || config.ReceiveDir != "/tmp/files" || config.DeviceName != "living-room-pc" {
		t.Fatalf("config = %+v", config)
	}
}

func TestLoadConfigFlagsOverrideEnvironment(t *testing.T) {
	environment := map[string]string{"DECKDROP_PORT": "9000", "DECKDROP_DEVICE_NAME": "from-env"}

	config, err := loadConfig(t, []string{"-port", "9100"}, environment)

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.Port != 9100 {
		t.Fatalf("port = %d, want the flag value 9100", config.Port)
	}
	if config.DeviceName != "from-env" {
		t.Fatalf("device name = %q, want the environment value", config.DeviceName)
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	tests := map[string]struct {
		args        []string
		environment map[string]string
	}{
		"non-numeric port in environment": {environment: map[string]string{"DECKDROP_PORT": "abc"}},
		"port too high":                   {args: []string{"-port", "70000"}},
		"port zero":                       {args: []string{"-port", "0"}},
		"negative port":                   {args: []string{"-port", "-1"}},
		"empty directory":                 {args: []string{"-dir", ""}},
		"empty device name":               {args: []string{"-name", ""}},
		"unknown flag":                    {args: []string{"-nope"}},
		"unexpected argument":             {args: []string{"stray"}},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			if _, err := loadConfig(t, test.args, test.environment); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadConfigHelpFlagReturnsErrHelpAndPrintsUsage(t *testing.T) {
	var output bytes.Buffer

	_, err := LoadConfig([]string{"-h"}, environmentOf(nil), &output)

	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp", err)
	}
	for _, want := range []string{"-port", "-dir", "-name", "DECKDROP_PORT"} {
		if !bytes.Contains(output.Bytes(), []byte(want)) {
			t.Errorf("usage does not mention %q:\n%s", want, output.String())
		}
	}
}
