package offer

import (
	"context"
	"testing"
	"time"
)

func newTestGrants() (*Grants, *time.Time) {
	now := time.Now()
	grants := NewGrants()
	grants.now = func() time.Time { return now }
	return grants, &now
}

func TestOnlyGrantedFilesAreAllowed(t *testing.T) {
	grants, _ := newTestGrants()
	grants.Allow("device-a", []File{{Name: "save.dat", Size: 10}})

	if size, ok := grants.Size("device-a", "save.dat"); !ok || size != 10 {
		t.Errorf("Size() = %d, %v, want 10, true", size, ok)
	}
	if _, ok := grants.Size("device-a", "other.dat"); ok {
		t.Error("a file that was not offered is allowed")
	}
	if _, ok := grants.Size("device-b", "save.dat"); ok {
		t.Error("another device can use the grant")
	}
}

func TestGrantEndsWhenComplete(t *testing.T) {
	grants, _ := newTestGrants()
	grants.Allow("device-a", []File{{Name: "save.dat", Size: 10}})
	grants.Complete("device-a", "save.dat")

	if _, ok := grants.Size("device-a", "save.dat"); ok {
		t.Error("grant outlived the completed upload")
	}
}

func TestUnusedGrantExpiresButUseKeepsItAlive(t *testing.T) {
	grants, now := newTestGrants()
	grants.Allow("device-a", []File{{Name: "keep.dat", Size: 1}, {Name: "drop.dat", Size: 1}})

	*now = now.Add(grantLifetime - time.Minute)
	grants.Size("device-a", "keep.dat")
	*now = now.Add(2 * time.Minute)

	if _, ok := grants.Size("device-a", "keep.dat"); !ok {
		t.Error("a grant in use expired")
	}
	if _, ok := grants.Size("device-a", "drop.dat"); ok {
		t.Error("an unused grant did not expire")
	}
}

func TestCoversNeedsEveryFileWithTheSameSize(t *testing.T) {
	grants, _ := newTestGrants()
	files := []File{{Name: "a", Size: 1}, {Name: "b", Size: 2}}
	grants.Allow("device-a", files)

	if !grants.Covers("device-a", files) {
		t.Error("Covers() = false for the granted files")
	}
	if grants.Covers("device-a", []File{{Name: "a", Size: 1}, {Name: "b", Size: 3}}) {
		t.Error("Covers() = true although a size changed")
	}
	if grants.Covers("device-a", nil) {
		t.Error("Covers() = true for no files")
	}
}

func TestPolicy(t *testing.T) {
	if !Policy(true).Decide(context.Background(), Offer{}) {
		t.Error("Policy(true) declined")
	}
	if Policy(false).Decide(context.Background(), Offer{}) {
		t.Error("Policy(false) accepted")
	}
}
