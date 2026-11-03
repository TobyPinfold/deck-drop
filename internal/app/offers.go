package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/TobyPinfold/deck-drop/internal/offer"
)

const (
	defaultOfferTimeout = time.Minute
	previewNames        = 3
)

// PendingOffer is a paired device asking to send files, waiting for the user
// to accept or decline.
type PendingOffer struct {
	ID     int      `json:"id"`
	Device string   `json:"device"`
	Count  int      `json:"count"`
	Bytes  int64    `json:"bytes"`
	Names  []string `json:"names"`

	answer chan bool
}

// Decide asks the user whether to accept the files offered by a paired
// device. It declines if nobody answers in time. Devices set to always
// accept are not asked.
func (app *App) Decide(ctx context.Context, request offer.Offer) bool {
	if request.From.AutoAccept {
		return true
	}

	pending := PendingOffer{
		Device: request.From.Name,
		Count:  len(request.Files),
		answer: make(chan bool, 1),
	}
	for index, file := range request.Files {
		pending.Bytes += file.Size
		if index < previewNames {
			pending.Names = append(pending.Names, file.Name)
		}
	}

	app.mu.Lock()
	app.lastOfferID++
	pending.ID = app.lastOfferID
	app.offers = append(app.offers, pending)
	snapshot := slices.Clone(app.offers)
	app.mu.Unlock()
	app.notifyOffers(snapshot)

	timeout := app.options.OfferTimeout
	if timeout <= 0 {
		timeout = defaultOfferTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	accepted := false
	select {
	case accepted = <-pending.answer:
	case <-timer.C:
	case <-ctx.Done():
	}

	app.mu.Lock()
	app.offers = slices.DeleteFunc(app.offers, func(other PendingOffer) bool { return other.ID == pending.ID })
	snapshot = slices.Clone(app.offers)
	app.mu.Unlock()
	app.notifyOffers(snapshot)
	return accepted
}

// Offers lists the files other devices are waiting to send.
func (app *App) Offers() []PendingOffer {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.offers)
}

// RespondToOffer accepts or declines a pending offer.
func (app *App) RespondToOffer(id int, accept bool) error {
	app.mu.Lock()
	index := slices.IndexFunc(app.offers, func(pending PendingOffer) bool { return pending.ID == id })
	if index < 0 {
		app.mu.Unlock()
		return fmt.Errorf("offer %d is no longer waiting", id)
	}
	answer := app.offers[index].answer
	app.mu.Unlock()

	select {
	case answer <- accept:
	default:
	}
	return nil
}

// SetAutoAccept chooses whether files from a paired device are accepted
// without asking.
func (app *App) SetAutoAccept(fingerprint string, enabled bool) error {
	store := app.currentConfig().Trust
	if store == nil {
		return errPairingUnavailable
	}
	return store.SetAutoAccept(fingerprint, enabled)
}

func (app *App) notifyOffers(snapshot []PendingOffer) {
	if app.options.OnOffersChanged != nil {
		app.options.OnOffersChanged(snapshot)
	}
}
