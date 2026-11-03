// Package discovery announces this device on the local network so other
// DeckDrop devices can find it.
package discovery

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/zeroconf/v2"
)

// ServiceType is the mDNS service type DeckDrop devices advertise.
const ServiceType = "_deckdrop._tcp"

const serviceDomain = "local."

// Registration is a live mDNS announcement.
type Registration struct {
	server    *zeroconf.Server
	closeOnce sync.Once
}

// Announce advertises an instance of the DeckDrop service on port.
// Call Close to stop the announcement.
func Announce(instance string, port int) (*Registration, error) {
	server, err := zeroconf.Register(InstanceName(instance), ServiceType, serviceDomain, port, []string{"path=/upload"}, nil)
	if err != nil {
		return nil, fmt.Errorf("register mDNS service: %w", err)
	}
	return &Registration{server: server}, nil
}

// Close stops the announcement. It is safe to call more than once.
func (registration *Registration) Close() error {
	registration.closeOnce.Do(registration.server.Shutdown)
	return nil
}

// MDNS announces devices over mDNS. It satisfies the backend's Announcer.
type MDNS struct{}

// Announce advertises instance on port and returns a Closer that stops it.
func (MDNS) Announce(instance string, port int) (io.Closer, error) {
	registration, err := Announce(instance, port)
	if err != nil {
		return nil, err
	}
	return registration, nil
}

// InstanceName is how a device name appears on the network: dots are
// replaced because other devices fail to resolve them in an instance name.
func InstanceName(instance string) string {
	return strings.ReplaceAll(instance, ".", "-")
}

// MDNSScanner finds devices by browsing for the DeckDrop service over mDNS.
type MDNSScanner struct {
	// Duration is how long each scan listens for announcements.
	Duration time.Duration
}

// Scan listens for Duration and returns the devices that announced
// themselves in that time.
func (scanner MDNSScanner) Scan(ctx context.Context) ([]Device, error) {
	ctx, cancel := context.WithTimeout(ctx, scanner.Duration)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	browseDone := make(chan error, 1)
	go func() {
		browseDone <- zeroconf.Browse(ctx, ServiceType, serviceDomain, entries)
	}()

	devices := []Device{}
	for {
		select {
		case entry, open := <-entries:
			if !open {
				entries = nil // Browse closes the channel when it finishes
				continue
			}
			if device, ok := deviceFromEntry(entry); ok {
				devices = append(devices, device)
			}
		case err := <-browseDone:
			if err != nil {
				return nil, fmt.Errorf("browse mDNS: %w", err)
			}
			return devices, nil
		}
	}
}

func deviceFromEntry(entry *zeroconf.ServiceEntry) (Device, bool) {
	var host string
	switch {
	case len(entry.AddrIPv4) > 0:
		host = entry.AddrIPv4[0].String()
	case len(entry.AddrIPv6) > 0:
		host = entry.AddrIPv6[0].String()
	default:
		return Device{}, false
	}
	return Device{Name: entry.Instance, Host: host, Port: entry.Port}, true
}
