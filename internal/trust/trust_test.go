package trust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPairingSurvivesReopening(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paired.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add(Peer{Fingerprint: "abc", Name: "deck"}); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Has("abc") {
		t.Error("paired device was lost after reopening")
	}
}

func TestRemoveRevokesAccess(t *testing.T) {
	store, _ := Open("")
	_ = store.Add(Peer{Fingerprint: "abc", Name: "deck"})

	if err := store.Remove("abc"); err != nil {
		t.Fatal(err)
	}
	if store.Has("abc") {
		t.Error("device is still paired after Remove")
	}
}

func TestEmptyFingerprintIsNeverPaired(t *testing.T) {
	store, _ := Open("")
	_ = store.Add(Peer{Fingerprint: "", Name: "odd"})
	if store.Has("") {
		t.Error("an empty fingerprint must never count as paired")
	}
}

func TestAddReplacesTheSameDevice(t *testing.T) {
	store, _ := Open("")
	_ = store.Add(Peer{Fingerprint: "abc", Name: "old"})
	_ = store.Add(Peer{Fingerprint: "abc", Name: "new"})

	peers := store.List()
	if len(peers) != 1 || peers[0].Name != "new" {
		t.Errorf("peers = %v, want one device named new", peers)
	}
}

func TestOnChangeReportsTheNewList(t *testing.T) {
	store, _ := Open("")
	var reported []Peer
	store.OnChange(func(peers []Peer) { reported = peers })

	_ = store.Add(Peer{Fingerprint: "abc", Name: "deck"})

	if len(reported) != 1 {
		t.Errorf("reported = %v, want the new device", reported)
	}
}

func TestOpenRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paired.json")
	store, _ := Open("")
	store.path = path
	if err := writeFile(path, "not json"); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Error("Open() error = nil, want a parse error")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestAutoAcceptIsKeptAcrossRepairing(t *testing.T) {
	store, _ := Open("")
	_ = store.Add(Peer{Fingerprint: "abc", Name: "deck"})
	_ = store.SetAutoAccept("abc", true)
	_ = store.Add(Peer{Fingerprint: "abc", Name: "renamed"})

	peer, ok := store.Get("abc")
	if !ok || !peer.AutoAccept || peer.Name != "renamed" {
		t.Errorf("Get() = %+v, %v, want the renamed device with auto accept kept", peer, ok)
	}
}
