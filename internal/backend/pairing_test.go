package backend

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

type receiverFixture struct {
	backend   *Backend
	address   string
	directory string
	identity  *identity.Identity
	trust     *trust.Store
	pairing   *pairing.Manager
}

func startPairingBackend(t *testing.T) receiverFixture {
	t.Helper()

	receiverIdentity, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	store, err := trust.Open("")
	if err != nil {
		t.Fatal(err)
	}
	manager := pairing.NewManager(receiverIdentity.Fingerprint())
	directory := t.TempDir()
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	running := New(logger, Config{
		ReceiveDir: directory, DeviceName: "receiver",
		Identity: receiverIdentity, Trust: store, Pairing: manager,
	})
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Stop() })

	return receiverFixture{
		backend: running, address: net.JoinHostPort("127.0.0.1", strconv.Itoa(running.Port())),
		directory: directory, identity: receiverIdentity, trust: store, pairing: manager,
	}
}

func sendFile(t *testing.T, sender transfer.Sender, address, fingerprint string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	return sender.Send(context.Background(), address, fingerprint, path, nil)
}

func TestUnpairedDeviceCannotUpload(t *testing.T) {
	fixture := startPairingBackend(t)
	sender, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}

	err = sendFile(t, transfer.Sender{Secure: true, Identity: sender}, fixture.address, "")

	if !errors.Is(err, transfer.ErrNotPaired) {
		t.Fatalf("Send() error = %v, want ErrNotPaired", err)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.directory, "note.txt")); statErr == nil {
		t.Error("file was written by an unpaired device")
	}
}

func TestDeviceWithoutACertificateCannotUpload(t *testing.T) {
	fixture := startPairingBackend(t)
	err := sendFile(t, transfer.Sender{Secure: true}, fixture.address, "")
	if !errors.Is(err, transfer.ErrNotPaired) {
		t.Fatalf("Send() error = %v, want ErrNotPaired", err)
	}
}

func TestPairedDeviceCanUploadUntilUnpaired(t *testing.T) {
	fixture := startPairingBackend(t)
	sender, _ := identity.Generate()
	code, err := fixture.pairing.NewCode()
	if err != nil {
		t.Fatal(err)
	}

	peer, err := pairing.Pair(context.Background(), sender, "laptop", fixture.address, code.Value)
	if err != nil {
		t.Fatalf("Pair() error = %v", err)
	}
	if peer.Fingerprint != fixture.identity.Fingerprint() || peer.Name != "receiver" {
		t.Errorf("peer = %+v, want the receiver", peer)
	}
	if !fixture.trust.Has(sender.Fingerprint()) {
		t.Fatal("receiver did not remember the sender")
	}

	paired := transfer.Sender{Secure: true, Identity: sender}
	if err := sendFile(t, paired, fixture.address, peer.Fingerprint); err != nil {
		t.Fatalf("Send() after pairing error = %v", err)
	}

	if err := fixture.trust.Remove(sender.Fingerprint()); err != nil {
		t.Fatal(err)
	}
	if err := sendFile(t, paired, fixture.address, peer.Fingerprint); !errors.Is(err, transfer.ErrNotPaired) {
		t.Errorf("Send() after unpairing error = %v, want ErrNotPaired", err)
	}
}

func TestWrongCodeDoesNotPair(t *testing.T) {
	fixture := startPairingBackend(t)
	sender, _ := identity.Generate()
	if _, err := fixture.pairing.NewCode(); err != nil {
		t.Fatal(err)
	}

	_, err := pairing.Pair(context.Background(), sender, "laptop", fixture.address, "000000x")
	if !errors.Is(err, pairing.ErrRejected) {
		t.Fatalf("Pair() error = %v, want ErrRejected", err)
	}
	if fixture.trust.Has(sender.Fingerprint()) {
		t.Error("a wrong code paired the device")
	}
}

func TestPairingWithoutAnActiveCodeFails(t *testing.T) {
	fixture := startPairingBackend(t)
	sender, _ := identity.Generate()

	_, err := pairing.Pair(context.Background(), sender, "laptop", fixture.address, "123456")
	if !errors.Is(err, pairing.ErrRejected) {
		t.Fatalf("Pair() error = %v, want ErrRejected", err)
	}
}

func TestPairingBruteForceIsStopped(t *testing.T) {
	fixture := startPairingBackend(t)
	sender, _ := identity.Generate()
	code, _ := fixture.pairing.NewCode()

	for range 10 {
		_, _ = pairing.Pair(context.Background(), sender, "laptop", fixture.address, "wrong")
	}

	if _, err := pairing.Pair(context.Background(), sender, "laptop", fixture.address, code.Value); err == nil {
		t.Error("the right code worked after too many wrong attempts")
	}
}
