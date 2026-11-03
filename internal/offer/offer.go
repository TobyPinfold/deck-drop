// Package offer lets a receiver decide whether to accept files before any are
// written. A sender makes an offer; once accepted, the receiver grants it
// permission to upload exactly those files.
package offer

import (
	"context"
	"sync"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/trust"
)

// File is one file in an offer.
type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Offer is a paired device asking to send files.
type Offer struct {
	From  trust.Peer
	Files []File
}

// Decider decides whether an offer is accepted. It returns false to decline,
// and must return false promptly once ctx is done.
type Decider interface {
	Decide(ctx context.Context, offer Offer) bool
}

// DeciderFunc adapts a function to a Decider.
type DeciderFunc func(ctx context.Context, offer Offer) bool

// Decide calls the function.
func (decide DeciderFunc) Decide(ctx context.Context, offer Offer) bool { return decide(ctx, offer) }

// Policy is the decider for a device with no prompt: it accepts everything
// from a paired device when acceptPaired is set, and otherwise declines.
func Policy(acceptPaired bool) Decider {
	return DeciderFunc(func(context.Context, Offer) bool { return acceptPaired })
}

const grantLifetime = 24 * time.Hour

type grantKey struct {
	fingerprint string
	name        string
}

// Grants remembers which files each device may upload. A grant lasts until
// the file is complete or it has been unused for a day, so a paused transfer
// can resume without asking again.
type Grants struct {
	now func() time.Time

	mu     sync.Mutex
	grants map[grantKey]grant
}

type grant struct {
	size    int64
	expires time.Time
}

// NewGrants returns an empty set of grants.
func NewGrants() *Grants {
	return &Grants{now: time.Now, grants: make(map[grantKey]grant)}
}

// Allow lets the device with fingerprint upload files.
func (grants *Grants) Allow(fingerprint string, files []File) {
	grants.mu.Lock()
	defer grants.mu.Unlock()
	expires := grants.now().Add(grantLifetime)
	for _, file := range files {
		grants.grants[grantKey{fingerprint, file.Name}] = grant{size: file.Size, expires: expires}
	}
}

// Size returns the size the device was granted for name, and whether it may
// upload it. Using a grant keeps it alive.
func (grants *Grants) Size(fingerprint, name string) (int64, bool) {
	grants.mu.Lock()
	defer grants.mu.Unlock()

	key := grantKey{fingerprint, name}
	found, ok := grants.grants[key]
	if !ok {
		return 0, false
	}
	now := grants.now()
	if !now.Before(found.expires) {
		delete(grants.grants, key)
		return 0, false
	}
	found.expires = now.Add(grantLifetime)
	grants.grants[key] = found
	return found.size, true
}

// Covers reports whether every file is already granted with the same size,
// as when a sender asks again after a reconnect.
func (grants *Grants) Covers(fingerprint string, files []File) bool {
	for _, file := range files {
		if size, ok := grants.Size(fingerprint, file.Name); !ok || size != file.Size {
			return false
		}
	}
	return len(files) > 0
}

// Complete ends the grant for name once it has been fully uploaded.
func (grants *Grants) Complete(fingerprint, name string) {
	grants.mu.Lock()
	defer grants.mu.Unlock()
	delete(grants.grants, grantKey{fingerprint, name})
}
