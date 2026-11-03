package pairing

import (
	"errors"
	"regexp"
	"testing"
	"time"
)

func newTestManager(t *testing.T) (*Manager, *time.Time) {
	t.Helper()
	now := time.Now()
	manager := NewManager("server")
	manager.now = func() time.Time { return now }
	return manager, &now
}

func TestCodeIsSixDigits(t *testing.T) {
	manager, _ := newTestManager(t)
	code, err := manager.NewCode()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code.Value) {
		t.Errorf("code = %q, want six digits", code.Value)
	}
}

func TestCorrectProofPairsOnce(t *testing.T) {
	manager, _ := newTestManager(t)
	code, _ := manager.NewCode()
	proof := Proof(code.Value, "client", "server")

	if err := manager.Verify("client", proof); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := manager.Verify("client", proof); !errors.Is(err, ErrNoCode) {
		t.Errorf("second Verify() error = %v, want ErrNoCode", err)
	}
}

func TestCodeExpires(t *testing.T) {
	manager, now := newTestManager(t)
	code, _ := manager.NewCode()
	*now = now.Add(codeLifetime)

	if err := manager.Verify("client", Proof(code.Value, "client", "server")); !errors.Is(err, ErrNoCode) {
		t.Errorf("Verify() error = %v, want ErrNoCode after expiry", err)
	}
}

func TestVerifyWithoutACodeFails(t *testing.T) {
	manager, _ := newTestManager(t)
	if err := manager.Verify("client", "anything"); !errors.Is(err, ErrNoCode) {
		t.Errorf("Verify() error = %v, want ErrNoCode", err)
	}
}

func TestWrongProofIsRejected(t *testing.T) {
	manager, _ := newTestManager(t)
	_, _ = manager.NewCode()
	if err := manager.Verify("client", "wrong"); !errors.Is(err, ErrWrongCode) {
		t.Errorf("Verify() error = %v, want ErrWrongCode", err)
	}
}

func TestTooManyWrongAttemptsDiscardTheCode(t *testing.T) {
	manager, _ := newTestManager(t)
	code, _ := manager.NewCode()
	for range maxAttempts {
		_ = manager.Verify("client", "wrong")
	}

	if err := manager.Verify("client", Proof(code.Value, "client", "server")); !errors.Is(err, ErrNoCode) {
		t.Errorf("Verify() with the right proof error = %v, want ErrNoCode after lockout", err)
	}
}

func TestProofIsBoundToTheClientCertificate(t *testing.T) {
	manager, _ := newTestManager(t)
	code, _ := manager.NewCode()

	// A proof made for a different client (or server) certificate is useless.
	if err := manager.Verify("intruder", Proof(code.Value, "client", "server")); !errors.Is(err, ErrWrongCode) {
		t.Errorf("Verify() error = %v, want ErrWrongCode", err)
	}
}

func TestNewCodeReplacesTheOldOne(t *testing.T) {
	manager, _ := newTestManager(t)
	first, _ := manager.NewCode()
	second, _ := manager.NewCode()
	if first.Value == second.Value {
		t.Skip("random codes collided")
	}
	if err := manager.Verify("client", Proof(first.Value, "client", "server")); !errors.Is(err, ErrWrongCode) {
		t.Errorf("old code Verify() error = %v, want ErrWrongCode", err)
	}
}
