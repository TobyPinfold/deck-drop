package backend

import "io"

// Announcer advertises a device on the local network. Closing the returned
// io.Closer stops the announcement.
type Announcer interface {
	Announce(instance string, port int) (io.Closer, error)
}
