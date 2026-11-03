package backend

import (
	"bytes"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
)

func newTestBackend(logBuffer *bytes.Buffer) *Backend {
	logger := slog.New(slog.NewTextHandler(logBuffer, nil))
	return New(logger, Config{Port: 0})
}

func startTestBackend(t *testing.T) *Backend {
	t.Helper()

	testBackend := newTestBackend(&bytes.Buffer{})
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })
	return testBackend
}

func requestTo(t *testing.T, testBackend *Backend, method, path string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, "http://"+testBackend.Addr()+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestStartAndStop(t *testing.T) {
	testBackend := newTestBackend(&bytes.Buffer{})

	if testBackend.Running() {
		t.Fatal("backend should not be running before Start")
	}
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !testBackend.Running() {
		t.Fatal("backend should be running after Start")
	}
	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if testBackend.Running() {
		t.Fatal("backend should not be running after Stop")
	}
}

func TestStartTwiceFails(t *testing.T) {
	testBackend := startTestBackend(t)

	if err := testBackend.Start(); err == nil {
		t.Fatal("second Start should return an error")
	}
}

func TestStopWhenNotRunningIsNoop(t *testing.T) {
	testBackend := newTestBackend(&bytes.Buffer{})

	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop on idle backend: %v", err)
	}
}

func TestLogsStartup(t *testing.T) {
	var logBuffer bytes.Buffer
	testBackend := newTestBackend(&logBuffer)
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	if !strings.Contains(logBuffer.String(), "backend started") {
		t.Fatalf("expected startup log, got %q", logBuffer.String())
	}
}

func TestStartFailsWhenPortIsTaken(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	takenPort := listener.Addr().(*net.TCPAddr).Port
	testBackend := New(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), Config{Port: takenPort})

	if err := testBackend.Start(); err == nil {
		_ = testBackend.Stop()
		t.Fatal("Start should fail when the port is in use")
	}
	if testBackend.Running() {
		t.Fatal("backend should not be running after a failed Start")
	}
}

func TestServesRequestsWhileRunning(t *testing.T) {
	testBackend := startTestBackend(t)

	response := requestTo(t, testBackend, http.MethodGet, "/health")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func TestStopReleasesPort(t *testing.T) {
	testBackend := startTestBackend(t)
	address := testBackend.Addr()

	if err := testBackend.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	connection, err := net.Dial("tcp", address)
	if err == nil {
		_ = connection.Close()
		t.Fatal("server should refuse connections after Stop")
	}
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	testBackend := startTestBackend(t)

	response := requestTo(t, testBackend, http.MethodGet, "/does-not-exist")

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

func TestWrongMethodReturnsMethodNotAllowed(t *testing.T) {
	testBackend := startTestBackend(t)

	response := requestTo(t, testBackend, http.MethodPost, "/health")

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
	}
}
