package backend

import "io"

// Announcer advertises a device on the local network. Closing the returned
// io.Closer stops the announcement. fingerprint is the device's certificate
// fingerprint, empty if it has none.
type Announcer interface {
	Announce(instance string, port int, fingerprint string) (io.Closer, error)
}
