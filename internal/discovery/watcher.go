package discovery

import (
	"context"
	"reflect"
	"sort"
	"time"
)

// missesBeforeRemoval is how many consecutive scans a device may be missing
// from before it is considered gone. mDNS packets get lost, so a single miss
// is not enough.
const missesBeforeRemoval = 2

// Device is another DeckDrop device on the network.
type Device struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Scanner finds the DeckDrop devices currently on the network.
type Scanner interface {
	Scan(ctx context.Context) ([]Device, error)
}

type trackedDevice struct {
	device Device
	misses int
}

// Watch scans immediately and then every interval until ctx is cancelled,
// calling onChange with the devices sorted by name whenever the set changes.
// A failed scan leaves the known devices untouched.
func Watch(ctx context.Context, scanner Scanner, interval time.Duration, onChange func([]Device)) {
	tracked := make(map[string]*trackedDevice)
	var reported []Device

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		found, err := scanner.Scan(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			update(tracked, found)
			if current := sortedDevices(tracked); !reflect.DeepEqual(current, reported) {
				reported = current
				onChange(current)
			}
		}
		timer.Reset(interval)
	}
}

func update(tracked map[string]*trackedDevice, found []Device) {
	seen := make(map[string]bool, len(found))
	for _, device := range found {
		seen[device.Name] = true
		tracked[device.Name] = &trackedDevice{device: device}
	}
	for name, entry := range tracked {
		if seen[name] {
			continue
		}
		entry.misses++
		if entry.misses >= missesBeforeRemoval {
			delete(tracked, name)
		}
	}
}

func sortedDevices(tracked map[string]*trackedDevice) []Device {
	devices := make([]Device, 0, len(tracked))
	for _, entry := range tracked {
		devices = append(devices, entry.device)
	}
	sort.Slice(devices, func(first, second int) bool { return devices[first].Name < devices[second].Name })
	return devices
}
