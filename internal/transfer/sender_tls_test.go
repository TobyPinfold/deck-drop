package transfer

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/identity"
)

type fixedPin string

func (pin fixedPin) Fingerprint(string) string { return string(pin) }

// newTLSServer serves the upload protocol over TLS using a fresh identity.
func newTLSServer(t *testing.T) (address, directory, fingerprint string) {
	t.Helper()

	deviceIdentity, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	plain, directory := newTestServer(t)

	server := httptest.NewUnstartedServer(plain.Config.Handler)
	server.TLS = deviceIdentity.ServerConfig()
	server.StartTLS()
	t.Cleanup(server.Close)
	return server.Listener.Addr().String(), directory, deviceIdentity.Fingerprint()
}

func TestSenderSendsOverTLSToPinnedDevice(t *testing.T) {
	address, directory, fingerprint := newTLSServer(t)
	path := writeSourceFile(t, "hello.txt", []byte("hello over tls"))

	sender := Sender{Secure: true, Pins: fixedPin(fingerprint)}
	if err := sender.Send(context.Background(), address, path, nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(directory, "hello.txt"))
	if err != nil || string(got) != "hello over tls" {
		t.Errorf("received = %q, %v", got, err)
	}
}

func TestSenderRejectsDeviceWithADifferentCertificate(t *testing.T) {
	address, directory, _ := newTLSServer(t)
	other, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	path := writeSourceFile(t, "secret.txt", []byte("secret"))

	sender := Sender{Secure: true, Pins: fixedPin(other.Fingerprint())}
	err = sender.Send(context.Background(), address, path, nil)

	if !errors.Is(err, identity.ErrFingerprintMismatch) {
		t.Fatalf("Send() error = %v, want a fingerprint mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(directory, "secret.txt")); statErr == nil {
		t.Error("file was uploaded to a device with the wrong certificate")
	}
}

func TestSecureSenderCannotUsePlainHTTPDevice(t *testing.T) {
	server, _ := newTestServer(t)
	path := writeSourceFile(t, "plain.txt", []byte("x"))

	sender := Sender{Secure: true}
	if err := sender.Send(context.Background(), server.Listener.Addr().String(), path, nil); err == nil {
		t.Error("Send() error = nil, want failure against a plain HTTP device")
	}
}
