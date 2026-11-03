package backend

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/TobyPinfold/deck-drop/internal/identity"
	"github.com/TobyPinfold/deck-drop/internal/offer"
	"github.com/TobyPinfold/deck-drop/internal/pairing"
	"github.com/TobyPinfold/deck-drop/internal/transfer"
)

// scriptedDecider answers every offer the same way and counts them.
type scriptedDecider struct {
	mu      sync.Mutex
	accept  bool
	offered [][]offer.File
}

func (decider *scriptedDecider) Decide(_ context.Context, request offer.Offer) bool {
	decider.mu.Lock()
	defer decider.mu.Unlock()
	decider.offered = append(decider.offered, request.Files)
	return decider.accept
}

func (decider *scriptedDecider) count() int {
	decider.mu.Lock()
	defer decider.mu.Unlock()
	return len(decider.offered)
}

// pairedSender pairs a new device with the fixture and returns a sender for it
// and the receiver's fingerprint to pin.
func pairedSender(t *testing.T, fixture receiverFixture) (transfer.Sender, *identity.Identity) {
	t.Helper()
	self, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	code, err := fixture.pairing.NewCode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pairing.Pair(context.Background(), self, "laptop", fixture.address, code.Value); err != nil {
		t.Fatalf("Pair() error = %v", err)
	}
	return transfer.Sender{Secure: true, Identity: self, ChunkSize: 4}, self
}

func writeSource(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNothingIsWrittenWithoutAnAcceptedOffer(t *testing.T) {
	fixture := startPairingBackend(t, &scriptedDecider{accept: true})
	sender, _ := pairedSender(t, fixture)

	err := sender.Send(context.Background(), fixture.address, fixture.identity.Fingerprint(), writeSource(t, "sneaky.txt", "data"), nil)

	if !errors.Is(err, transfer.ErrNotAccepted) {
		t.Fatalf("Send() without an offer error = %v, want ErrNotAccepted", err)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.directory, "sneaky.txt")); statErr == nil {
		t.Error("a file was written without an accepted offer")
	}
}

func TestAcceptedOfferAllowsOnlyTheOfferedFiles(t *testing.T) {
	decider := &scriptedDecider{accept: true}
	fixture := startPairingBackend(t, decider)
	sender, _ := pairedSender(t, fixture)
	fingerprint := fixture.identity.Fingerprint()
	offered := writeSource(t, "offered.txt", "12345678")
	other := writeSource(t, "other.txt", "x")

	if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{offered}); err != nil {
		t.Fatalf("Offer() error = %v", err)
	}
	if err := sender.Send(context.Background(), fixture.address, fingerprint, offered, nil); err != nil {
		t.Fatalf("Send() of the offered file error = %v", err)
	}
	if err := sender.Send(context.Background(), fixture.address, fingerprint, other, nil); !errors.Is(err, transfer.ErrNotAccepted) {
		t.Errorf("Send() of a file that was not offered error = %v, want ErrNotAccepted", err)
	}
	got, _ := os.ReadFile(filepath.Join(fixture.directory, "offered.txt"))
	if string(got) != "12345678" {
		t.Errorf("received = %q", got)
	}
}

func TestDeclinedOfferIsEnforcedByTheBackend(t *testing.T) {
	fixture := startPairingBackend(t, &scriptedDecider{accept: false})
	sender, _ := pairedSender(t, fixture)
	fingerprint := fixture.identity.Fingerprint()
	path := writeSource(t, "no.txt", "data")

	if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{path}); !errors.Is(err, transfer.ErrDeclined) {
		t.Fatalf("Offer() error = %v, want ErrDeclined", err)
	}
	if err := sender.Send(context.Background(), fixture.address, fingerprint, path, nil); !errors.Is(err, transfer.ErrNotAccepted) {
		t.Errorf("Send() after a declined offer error = %v, want ErrNotAccepted", err)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.directory, "no.txt")); statErr == nil {
		t.Error("a declined file was written")
	}
}

func TestResumingAnAcceptedTransferDoesNotAskAgain(t *testing.T) {
	decider := &scriptedDecider{accept: true}
	fixture := startPairingBackend(t, decider)
	sender, _ := pairedSender(t, fixture)
	fingerprint := fixture.identity.Fingerprint()
	path := writeSource(t, "big.bin", "0123456789")

	if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{path}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_ = sender.Send(ctx, fixture.address, fingerprint, path, func(sent, _ int64) {
		if sent >= 4 {
			cancel()
		}
	})
	cancel()

	// Pausing and resuming, or offering the same file again, keeps the answer.
	if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{path}); err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), fixture.address, fingerprint, path, nil); err != nil {
		t.Fatalf("resumed Send() error = %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(fixture.directory, "big.bin"))
	if string(got) != "0123456789" {
		t.Errorf("received = %q, want the whole file", got)
	}
	if decider.count() != 1 {
		t.Errorf("the user was asked %d times, want once", decider.count())
	}
}

func TestAnotherSendOfTheSameFileAsksAgainOnceComplete(t *testing.T) {
	decider := &scriptedDecider{accept: true}
	fixture := startPairingBackend(t, decider)
	sender, _ := pairedSender(t, fixture)
	fingerprint := fixture.identity.Fingerprint()
	path := writeSource(t, "once.txt", "data")

	for range 2 {
		if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{path}); err != nil {
			t.Fatal(err)
		}
		if err := sender.Send(context.Background(), fixture.address, fingerprint, path, nil); err != nil {
			t.Fatal(err)
		}
	}
	if decider.count() != 2 {
		t.Errorf("the user was asked %d times, want twice", decider.count())
	}
}

func TestChunkBeyondTheOfferedSizeIsRejected(t *testing.T) {
	fixture := startPairingBackend(t, &scriptedDecider{accept: true})
	sender, self := pairedSender(t, fixture)
	fingerprint := fixture.identity.Fingerprint()
	path := writeSource(t, "small.txt", "1234")
	if err := sender.Offer(context.Background(), fixture.address, fingerprint, []string{path}); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: identity.ClientConfig(fingerprint, self)}}
	request, _ := http.NewRequest(http.MethodPost, "https://"+fixture.address+"/upload", bytes.NewReader([]byte("12345678")))
	request.Header.Set("X-File-Name", "small.txt")
	request.Header.Set("X-File-Offset", "0")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a chunk past the offered size", response.StatusCode)
	}
}

func TestHeadlessPolicyDeclinesUnlessToldToAccept(t *testing.T) {
	for _, acceptPaired := range []bool{false, true} {
		decider := offer.Policy(acceptPaired)
		fixture := startPairingBackend(t, decider)
		sender, _ := pairedSender(t, fixture)
		err := sender.Offer(context.Background(), fixture.address, fixture.identity.Fingerprint(), []string{writeSource(t, "p.txt", "x")})
		if acceptPaired != (err == nil) {
			t.Errorf("acceptPaired=%v: Offer() error = %v", acceptPaired, err)
		}
	}
}
