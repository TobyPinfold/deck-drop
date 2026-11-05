package version

import "testing"

func TestStringTrimsTheFile(t *testing.T) {
	original := raw
	t.Cleanup(func() { raw = original })

	raw = "0.1.0-rc.5\n"
	if got := String(); got != "0.1.0-rc.5" {
		t.Fatalf("String() = %q, want %q", got, "0.1.0-rc.5")
	}
}

func TestStringIsDevWhenEmpty(t *testing.T) {
	original := raw
	t.Cleanup(func() { raw = original })

	raw = "  \n"
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want %q", got, "dev")
	}
}
