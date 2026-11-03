package identity

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrCreateKeepsTheSameIdentity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "config")

	first, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}
	second, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreate() error = %v", err)
	}

	if first.Fingerprint() == "" || first.Fingerprint() != second.Fingerprint() {
		t.Errorf("fingerprints = %q and %q, want the same non-empty value", first.Fingerprint(), second.Fingerprint())
	}
}

func TestGenerateMakesDistinctIdentities(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() == second.Fingerprint() {
		t.Error("two generated identities share a fingerprint")
	}
}

func TestLoadOrCreateKeepsTheKeyPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not enforced on Windows")
	}
	dir := t.TempDir()
	if _, err := LoadOrCreate(dir); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, keyFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestLoadOrCreateRejectsACorruptIdentity(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{certFileName, keyFileName} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("garbage"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadOrCreate(dir); err == nil {
		t.Error("LoadOrCreate() error = nil, want an error rather than overwriting the identity")
	}
}
