package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/zeroconf/v2"
)

const lookupTimeout = 5 * time.Second

// browseForInstance returns the announced entry with the given instance name,
// or nil if it is not seen before the timeout.
func browseForInstance(t *testing.T, instance string) *zeroconf.ServiceEntry {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		_ = zeroconf.Browse(ctx, ServiceType, "local.", entries)
	}()

	for {
		select {
		case entry := <-entries:
			if entry != nil && entry.Instance == instance {
				return entry
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func TestAnnounceMakesServiceDiscoverable(t *testing.T) {
	const instance = "deckdrop-test-discoverable"
	const port = 45678

	registration, err := Announce(instance, port)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}
	t.Cleanup(func() { _ = registration.Close() })

	entry := browseForInstance(t, instance)

	if entry == nil {
		t.Fatal("announced service was not discovered")
	}
	if entry.Port != port {
		t.Fatalf("port = %d, want %d", entry.Port, port)
	}
}

func TestCloseStopsAnnouncement(t *testing.T) {
	const instance = "deckdrop-test-closed"

	registration, err := Announce(instance, 45679)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}
	if err := registration.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if entry := browseForInstance(t, instance); entry != nil {
		t.Fatal("service should not be discoverable after Close")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	registration, err := Announce("deckdrop-test-idempotent", 45680)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}

	if err := registration.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := registration.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMDNSAnnouncerAnnouncesInstance(t *testing.T) {
	const instance = "deckdrop-test-announcer"
	const port = 45681

	closer, err := MDNS{}.Announce(instance, port)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	if entry := browseForInstance(t, instance); entry == nil || entry.Port != port {
		t.Fatalf("announced service not discovered correctly: %+v", entry)
	}
}

func TestAnnounceMakesDottedInstanceNameDiscoverable(t *testing.T) {
	const port = 45682

	registration, err := Announce("my.deck.local", port)
	if err != nil {
		t.Fatalf("Announce: %v", err)
	}
	t.Cleanup(func() { _ = registration.Close() })

	if entry := browseForInstance(t, "my-deck-local"); entry == nil {
		t.Fatal("dotted instance name should be announced with dots replaced")
	}
}
