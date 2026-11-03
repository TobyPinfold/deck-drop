package backend

import (
	"bytes"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

func statusRequest(t *testing.T, testBackend *Backend, method, name string) *http.Response {
	t.Helper()

	query := url.Values{"filename": {name}}
	return requestTo(t, testBackend, method, "/status?"+query.Encode())
}

func receivedOffset(t *testing.T, response *http.Response) int {
	t.Helper()

	offset, err := strconv.Atoi(response.Header.Get(fileOffsetHeader))
	if err != nil {
		t.Fatalf("invalid %s header %q: %v", fileOffsetHeader, response.Header.Get(fileOffsetHeader), err)
	}
	return offset
}

func TestStatusOfNewFileIsZero(t *testing.T) {
	testBackend := startTestBackend(t)

	response := statusRequest(t, testBackend, http.MethodGet, "new.bin")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if offset := receivedOffset(t, response); offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
}

func TestStatusReportsBytesReceivedSoFar(t *testing.T) {
	testBackend := startTestBackend(t)
	uploadChunk(t, testBackend, "partial.bin", 0, randomBytes(t, 250))

	response := statusRequest(t, testBackend, http.MethodGet, "partial.bin")

	if offset := receivedOffset(t, response); offset != 250 {
		t.Fatalf("offset = %d, want 250", offset)
	}
}

func TestStatusSupportsHeadRequests(t *testing.T) {
	testBackend := startTestBackend(t)
	uploadChunk(t, testBackend, "partial.bin", 0, randomBytes(t, 250))

	response := statusRequest(t, testBackend, http.MethodHead, "partial.bin")

	if response.StatusCode != http.StatusOK || receivedOffset(t, response) != 250 {
		t.Fatalf("status = %d, offset header = %q", response.StatusCode, response.Header.Get(fileOffsetHeader))
	}
}

func TestInterruptedUploadResumesFromReportedOffset(t *testing.T) {
	testBackend := startTestBackend(t)
	content := randomBytes(t, 1000)
	uploadChunk(t, testBackend, "resumed.bin", 0, content[:600])

	resumeOffset := receivedOffset(t, statusRequest(t, testBackend, http.MethodGet, "resumed.bin"))
	uploadChunk(t, testBackend, "resumed.bin", resumeOffset, content[resumeOffset:])

	if !bytes.Equal(receivedFile(t, testBackend, "resumed.bin"), content) {
		t.Fatal("resumed file differs from the original content")
	}
}

func TestStatusRejectsMissingOrInvalidFilename(t *testing.T) {
	for _, name := range []string{"", "../secret.bin", "nested/file.bin"} {
		t.Run(name, func(t *testing.T) {
			testBackend := startTestBackend(t)

			response := statusRequest(t, testBackend, http.MethodGet, name)

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func TestStatusWithWrongMethodReturnsMethodNotAllowed(t *testing.T) {
	testBackend := startTestBackend(t)

	response := statusRequest(t, testBackend, http.MethodPost, "file.bin")

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
	}
}
