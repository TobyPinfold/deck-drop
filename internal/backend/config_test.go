package backend

import "testing"

func TestDefaultConfigUsesPort8080(t *testing.T) {
	config := DefaultConfig()

	if config.Port != 8080 {
		t.Fatalf("default port = %d, want 8080", config.Port)
	}
}
