package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/offer"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
	"github.com/TobyPinfold/deck-drop/internal/trust"
)

// offeringSender answers offers with a fixed error and records them.
type offeringSender struct {
	fakeSender
	offerErr error
	mu2      sync.Mutex
	offers   int
}

func (sender *offeringSender) Offer(_ context.Context, _, _ string, _ []string) error {
	sender.mu2.Lock()
	defer sender.mu2.Unlock()
	sender.offers++
	return sender.offerErr
}

func TestAcceptedOfferSendsTheFiles(t *testing.T) {
	sender := &offeringSender{}
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatal(err)
	}
	final := waitForFinished(t, transfers)

	if final[0].Status != StatusDone {
		t.Errorf("status = %q, want done", final[0].Status)
	}
	if len(sender.sentFiles()) != 1 {
		t.Errorf("sent = %v", sender.sentFiles())
	}
}

func TestDeclinedOfferSendsNothing(t *testing.T) {
	sender := &offeringSender{offerErr: transfer.ErrDeclined}
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(eventTimeout)
	for {
		select {
		case list := <-transfers:
			if list[0].Status == StatusDeclined {
				if len(sender.sentFiles()) != 0 {
					t.Errorf("declined files were sent: %v", sender.sentFiles())
				}
				return
			}
		case <-deadline:
			t.Fatal("transfer never became declined")
		}
	}
}

func TestResumingDoesNotOfferAgain(t *testing.T) {
	sender := &offeringSender{}
	sender.failed = map[string]bool{"/games/a.sav": true}
	app, transfers := newSendingApp(t, sender)

	if err := app.SendFiles(otherDevice.Name, []string{"/games/a.sav"}); err != nil {
		t.Fatal(err)
	}
	waitForFinished(t, transfers)
	sender.failed = nil
	if err := app.Resume(1); err != nil {
		t.Fatal(err)
	}
	waitForFinished(t, transfers)

	sender.mu2.Lock()
	defer sender.mu2.Unlock()
	if sender.offers != 1 {
		t.Errorf("offered %d times, want once", sender.offers)
	}
}

func newPendingOffer(app *App, request offer.Offer) <-chan bool {
	answer := make(chan bool, 1)
	go func() { answer <- app.Decide(context.Background(), request) }()
	return answer
}

func waitForPending(t *testing.T, app *App) PendingOffer {
	t.Helper()
	deadline := time.After(eventTimeout)
	for {
		if pending := app.Offers(); len(pending) > 0 {
			return pending[0]
		}
		select {
		case <-deadline:
			t.Fatal("no offer became pending")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestPendingOfferIsAnsweredByTheUser(t *testing.T) {
	for _, accept := range []bool{true, false} {
		app, _ := newSendingApp(t, &fakeSender{})
		answer := newPendingOffer(app, offer.Offer{
			From:  trust.Peer{Name: "laptop"},
			Files: []offer.File{{Name: "a.txt", Size: 5}, {Name: "b.txt", Size: 7}},
		})

		pending := waitForPending(t, app)
		if pending.Device != "laptop" || pending.Count != 2 || pending.Bytes != 12 {
			t.Errorf("pending = %+v", pending)
		}
		if err := app.RespondToOffer(pending.ID, accept); err != nil {
			t.Fatal(err)
		}
		if got := <-answer; got != accept {
			t.Errorf("Decide() = %v, want %v", got, accept)
		}
		if len(app.Offers()) != 0 {
			t.Error("answered offer is still pending")
		}
	}
}

func TestUnansweredOfferIsDeclined(t *testing.T) {
	app, _ := newSendingApp(t, &fakeSender{})
	app.options.OfferTimeout = 10 * time.Millisecond

	if app.Decide(context.Background(), offer.Offer{From: trust.Peer{Name: "laptop"}, Files: []offer.File{{Name: "a"}}}) {
		t.Error("an unanswered offer was accepted")
	}
}

func TestAutoAcceptSkipsThePrompt(t *testing.T) {
	app, _ := newSendingApp(t, &fakeSender{})

	if !app.Decide(context.Background(), offer.Offer{From: trust.Peer{AutoAccept: true}, Files: []offer.File{{Name: "a"}}}) {
		t.Error("a device set to always accept was asked")
	}
	if len(app.Offers()) != 0 {
		t.Error("an offer was shown")
	}
}
