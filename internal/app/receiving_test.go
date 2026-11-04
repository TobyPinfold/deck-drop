package app

import (
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/trust"
)

func TestIncomingFilesAppearAsDownloadsThatFinish(t *testing.T) {
	store, err := trust.Open("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add(trust.Peer{Fingerprint: "deck-fp", Name: "steam-deck"}); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	app.config.Trust = store

	app.Receiving("deck-fp", "save.sav", 40, 100)
	downloads := app.Transfers()
	if len(downloads) != 1 {
		t.Fatalf("Transfers() = %+v, want one download", downloads)
	}
	got := downloads[0]
	if got.Direction != DirectionReceive || got.Device != "steam-deck" || got.Status != StatusReceiving || got.Sent != 40 || got.Total != 100 {
		t.Fatalf("download = %+v, want receiving 40 of 100 from steam-deck", got)
	}

	app.ReceiveStopped("deck-fp", "save.sav")
	if status := app.Transfers()[0].Status; status != StatusInterrupted {
		t.Fatalf("status after the sender stopped = %q, want %q", status, StatusInterrupted)
	}

	app.Receiving("deck-fp", "save.sav", 100, 100)
	downloads = app.Transfers()
	if len(downloads) != 1 || downloads[0].Status != StatusDone {
		t.Fatalf("Transfers() after the last chunk = %+v, want the same download done", downloads)
	}
}
