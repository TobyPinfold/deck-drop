package backend

import (
	"bytes"
	"crypto/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func postChunk(t *testing.T, testBackend *Backend, name, offset string, chunk []byte) *http.Response {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, "http://"+testBackend.Addr()+"/upload", bytes.NewReader(chunk))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if name != "" {
		request.Header.Set("X-File-Name", name)
	}
	if offset != "" {
		request.Header.Set("X-File-Offset", offset)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /upload: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func uploadChunk(t *testing.T, testBackend *Backend, name string, offset int, chunk []byte) {
	t.Helper()

	response := postChunk(t, testBackend, name, strconv.Itoa(offset), chunk)

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("upload %s at %d: status = %d, want %d", name, offset, response.StatusCode, http.StatusNoContent)
	}
}

func receivedFile(t *testing.T, testBackend *Backend, name string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(testBackend.config.ReceiveDir, name))
	if err != nil {
		t.Fatalf("read received file %s: %v", name, err)
	}
	return content
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()

	content := make([]byte, size)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return content
}

func TestMultiChunkUploadProducesIdenticalFile(t *testing.T) {
	testBackend := startTestBackend(t)
	content := randomBytes(t, 1000)

	uploadChunk(t, testBackend, "save.tar.gz", 0, content[:400])
	uploadChunk(t, testBackend, "save.tar.gz", 400, content[400:800])
	uploadChunk(t, testBackend, "save.tar.gz", 800, content[800:])

	if !bytes.Equal(receivedFile(t, testBackend, "save.tar.gz"), content) {
		t.Fatal("received file differs from the uploaded content")
	}
}

func TestChunksOfDifferentFilesDoNotInterfere(t *testing.T) {
	testBackend := startTestBackend(t)
	firstContent := randomBytes(t, 600)
	secondContent := randomBytes(t, 600)

	uploadChunk(t, testBackend, "first.bin", 0, firstContent[:300])
	uploadChunk(t, testBackend, "second.bin", 0, secondContent[:300])
	uploadChunk(t, testBackend, "second.bin", 300, secondContent[300:])
	uploadChunk(t, testBackend, "first.bin", 300, firstContent[300:])

	if !bytes.Equal(receivedFile(t, testBackend, "first.bin"), firstContent) {
		t.Fatal("first.bin differs from the uploaded content")
	}
	if !bytes.Equal(receivedFile(t, testBackend, "second.bin"), secondContent) {
		t.Fatal("second.bin differs from the uploaded content")
	}
}

func TestUploadRejectsBadRequests(t *testing.T) {
	tests := map[string]struct {
		name   string
		offset string
	}{
		"missing name":       {name: "", offset: "0"},
		"missing offset":     {name: "file.bin", offset: ""},
		"non-numeric offset": {name: "file.bin", offset: "abc"},
		"negative offset":    {name: "file.bin", offset: "-1"},
		"path traversal":     {name: "../escape.bin", offset: "0"},
		"nested path":        {name: "nested/file.bin", offset: "0"},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			testBackend := startTestBackend(t)

			response := postChunk(t, testBackend, test.name, test.offset, []byte("data"))

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
			}
			entries, err := os.ReadDir(testBackend.config.ReceiveDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("nothing should be written, got %v (err: %v)", entries, err)
			}
		})
	}
}

func TestUploadWithWrongMethodReturnsMethodNotAllowed(t *testing.T) {
	testBackend := startTestBackend(t)

	response := requestTo(t, testBackend, http.MethodGet, "/upload")

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestStartFailsWhenReceiveDirCannotBeCreated(t *testing.T) {
	blockingFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockingFile, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	testBackend := newTestBackend(t, &bytes.Buffer{})
	testBackend.config.ReceiveDir = filepath.Join(blockingFile, "received")

	if err := testBackend.Start(); err == nil {
		_ = testBackend.Stop()
		t.Fatal("Start should fail when the receive directory cannot be created")
	}
	if testBackend.Running() {
		t.Fatal("backend should not be running after a failed Start")
	}
}
