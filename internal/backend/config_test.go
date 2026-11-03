package backend

import "testing"

func TestDefaultConfigUsesPort8080(t *testing.T) {
	config := DefaultConfig()

	if config.Port != 8080 {
		t.Fatalf("default port = %d, want 8080", config.Port)
	}
}

func TestDefaultConfigHasDeviceName(t *testing.T) {
	config := DefaultConfig()

	if config.DeviceName == "" {
		t.Fatal("default device name should not be empty")
	}
}
