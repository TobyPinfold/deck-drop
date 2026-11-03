package backend

import (
	"bytes"
	"crypto/tls"
	"log/slog"
	"net/http"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/identity"
)

func startTLSBackend(t *testing.T, announcer Announcer) (*Backend, *identity.Identity) {
	t.Helper()

	deviceIdentity, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	testBackend := New(logger, Config{ReceiveDir: t.TempDir(), DeviceName: "deck", Identity: deviceIdentity, Announcer: announcer})
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })
	return testBackend, deviceIdentity
}

func TestBackendWithIdentityServesOnlyTLS(t *testing.T) {
	testBackend, deviceIdentity := startTLSBackend(t, nil)

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: identity.ClientConfig(deviceIdentity.Fingerprint())}}
	response, err := client.Get("https://" + testBackend.Addr() + "/health")
	if err != nil {
		t.Fatalf("GET over TLS: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", response.StatusCode)
	}

	if plain, err := http.Get("http://" + testBackend.Addr() + "/health"); err == nil {
		_ = plain.Body.Close()
		if plain.StatusCode == http.StatusOK {
			t.Error("plain HTTP request succeeded, want it refused")
		}
	}
}

func TestBackendRejectsOldTLSVersions(t *testing.T) {
	testBackend, _ := startTLSBackend(t, nil)

	connection, err := tls.Dial("tcp", testBackend.Addr(), &tls.Config{InsecureSkipVerify: true, MaxVersion: tls.VersionTLS12}) // #nosec G402 -- test client
	if err == nil {
		_ = connection.Close()
		t.Error("TLS 1.2 handshake succeeded, want TLS 1.3 only")
	}
}

func TestBackendAnnouncesItsFingerprint(t *testing.T) {
	announcer := &fakeAnnouncer{}
	_, deviceIdentity := startTLSBackend(t, announcer)

	if len(announcer.fingerprints) != 1 || announcer.fingerprints[0] != deviceIdentity.Fingerprint() {
		t.Errorf("announced fingerprints = %v, want [%s]", announcer.fingerprints, deviceIdentity.Fingerprint())
	}
}
