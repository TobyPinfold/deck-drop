// Package security sets up a device's identity and the devices it trusts.
package security

import (
	"fmt"

	"github.com/TobyPinfold/deck-drop/internal/backend"
	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

// Setup returns config with this device's certificate, its paired devices
// and pairing enabled, loading or creating them in the user's config folder.
func Setup(config backend.Config) (backend.Config, error) {
	identityDir, err := identity.DefaultDir()
	if err != nil {
		return config, err
	}
	deviceIdentity, err := identity.LoadOrCreate(identityDir)
	if err != nil {
		return config, fmt.Errorf("device certificate: %w", err)
	}

	pairedPath, err := trust.DefaultPath()
	if err != nil {
		return config, err
	}
	paired, err := trust.Open(pairedPath)
	if err != nil {
		return config, err
	}

	config.Identity = deviceIdentity
	config.Trust = paired
	config.Pairing = pairing.NewManager(deviceIdentity.Fingerprint())
	return config, nil
}
