// Package discovery announces this device on the local network so other
// DeckDrop devices can find it.
package discovery

import (
	"fmt"
	"sync"

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
	server, err := zeroconf.Register(instance, ServiceType, serviceDomain, port, []string{"path=/upload"}, nil)
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
