package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type scanResult struct {
	devices []Device
	err     error
}

// scriptedScanner returns the scripted results in order and cancels the
// context once they are used up, so Watch returns on its own.
type scriptedScanner struct {
	results []scanResult
	next    int
	cancel  context.CancelFunc
}

func (scanner *scriptedScanner) Scan(context.Context) ([]Device, error) {
	if scanner.next >= len(scanner.results) {
		scanner.cancel()
		return nil, context.Canceled
	}
	result := scanner.results[scanner.next]
	scanner.next++
	return result.devices, result.err
}

// watchScript runs Watch over the scripted scans and returns every list of
// devices reported to onChange.
func watchScript(t *testing.T, results ...scanResult) [][]Device {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var reported [][]Device
	Watch(ctx, &scriptedScanner{results: results, cancel: cancel}, time.Millisecond, func(devices []Device) {
		reported = append(reported, devices)
	})
	return reported
}

func seen(devices ...Device) scanResult { return scanResult{devices: devices} }

var (
	deck   = Device{Name: "steam-deck", Host: "192.168.1.20", Port: 8080}
	laptop = Device{Name: "my-laptop", Host: "192.168.1.21", Port: 8080}
)

func TestWatchReportsNewDevicesSortedByName(t *testing.T) {
	reported := watchScript(t, seen(deck, laptop))

	want := [][]Device{{laptop, deck}}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported = %v, want %v", reported, want)
	}
}

func TestWatchOnlyReportsChanges(t *testing.T) {
	reported := watchScript(t, seen(deck), seen(deck), seen(deck))

	if len(reported) != 1 {
		t.Fatalf("reported %d times, want once: %v", len(reported), reported)
	}
}

func TestWatchRemovesADeviceAfterItIsMissedTwice(t *testing.T) {
	reported := watchScript(t, seen(deck), seen(), seen())

	want := [][]Device{{deck}, {}}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported = %v, want %v", reported, want)
	}
}

func TestWatchKeepsADeviceThatReappearsInTime(t *testing.T) {
	reported := watchScript(t, seen(deck), seen(), seen(deck), seen())

	if len(reported) != 1 {
		t.Fatalf("reported = %v, want the device kept", reported)
	}
}

func TestWatchKeepsDevicesWhenAScanFails(t *testing.T) {
	failed := scanResult{err: errors.New("network down")}

	reported := watchScript(t, seen(deck), failed, failed, failed)

	if len(reported) != 1 {
		t.Fatalf("reported = %v, want devices kept through scan errors", reported)
	}
}

func TestWatchUpdatesADeviceThatChangesAddress(t *testing.T) {
	moved := Device{Name: deck.Name, Host: "192.168.1.99", Port: deck.Port}

	reported := watchScript(t, seen(deck), seen(moved))

	want := [][]Device{{deck}, {moved}}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported = %v, want %v", reported, want)
	}
}
