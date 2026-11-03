package backend

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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

func TestUploadOverMaxChunkSizeIsRejected(t *testing.T) {
	testBackend := newTestBackend(t, &bytes.Buffer{})
	testBackend.config.MaxChunkSize = 100
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	response := postChunk(t, testBackend, "big.bin", "0", randomBytes(t, 101))

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusRequestEntityTooLarge)
	}
}

func TestUploadAtMaxChunkSizeIsAccepted(t *testing.T) {
	testBackend := newTestBackend(t, &bytes.Buffer{})
	testBackend.config.MaxChunkSize = 100
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })
	content := randomBytes(t, 100)

	uploadChunk(t, testBackend, "exact.bin", 0, content)

	if !bytes.Equal(receivedFile(t, testBackend, "exact.bin"), content) {
		t.Fatal("received file differs from the uploaded content")
	}
}

func TestDefaultMaxChunkSizeAppliesWhenNotConfigured(t *testing.T) {
	config := Config{}

	if config.maxChunkSize() != defaultMaxChunkSize {
		t.Fatalf("maxChunkSize = %d, want %d", config.maxChunkSize(), defaultMaxChunkSize)
	}
}

func TestStreamedUploadOverMaxChunkSizeIsRejected(t *testing.T) {
	testBackend := newTestBackend(t, &bytes.Buffer{})
	testBackend.config.MaxChunkSize = 100
	if err := testBackend.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = testBackend.Stop() })

	// Hiding the reader's length makes the client send a chunked body with no
	// Content-Length, so the size can only be enforced while reading.
	body := io.NopCloser(bytes.NewReader(randomBytes(t, 500)))
	request, err := http.NewRequest(http.MethodPost, "http://"+testBackend.Addr()+"/upload", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	request.Header.Set(fileNameHeader, "streamed.bin")
	request.Header.Set(fileOffsetHeader, "0")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /upload: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusRequestEntityTooLarge)
	}
}

func TestLargeFileUploadedInManyChunksIsIdentical(t *testing.T) {
	const chunkSize = 64 * 1024
	testBackend := startTestBackend(t)
	content := randomBytes(t, 5*1024*1024+123)

	for offset := 0; offset < len(content); offset += chunkSize {
		uploadChunk(t, testBackend, "large.bin", offset, content[offset:min(offset+chunkSize, len(content))])
	}

	if !bytes.Equal(receivedFile(t, testBackend, "large.bin"), content) {
		t.Fatal("large file differs from the uploaded content")
	}
}

func TestConcurrentChunkUploadsProduceIdenticalFiles(t *testing.T) {
	const (
		fileCount     = 4
		chunksPerFile = 8
		chunkSize     = 4096
	)
	testBackend := startTestBackend(t)
	contents := make([][]byte, fileCount)
	for file := range contents {
		contents[file] = randomBytes(t, chunksPerFile*chunkSize)
	}

	var uploads sync.WaitGroup
	for file := range contents {
		for chunk := range chunksPerFile {
			uploads.Add(1)
			go func() {
				defer uploads.Done()
				offset := chunk * chunkSize
				uploadChunk(t, testBackend, fmt.Sprintf("file-%d.bin", file), offset, contents[file][offset:offset+chunkSize])
			}()
		}
	}
	uploads.Wait()

	for file, content := range contents {
		if !bytes.Equal(receivedFile(t, testBackend, fmt.Sprintf("file-%d.bin", file)), content) {
			t.Errorf("file-%d.bin differs from the uploaded content", file)
		}
	}
}

func TestUploadCanResumeAfterBackendRestart(t *testing.T) {
	receiveDir := t.TempDir()
	content := randomBytes(t, 2000)

	startBackend := func() *Backend {
		testBackend := newTestBackend(t, &bytes.Buffer{})
		testBackend.config.ReceiveDir = receiveDir
		if err := testBackend.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}
		t.Cleanup(func() { _ = testBackend.Stop() })
		return testBackend
	}

	firstRun := startBackend()
	uploadChunk(t, firstRun, "interrupted.bin", 0, content[:1200])
	if err := firstRun.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	secondRun := startBackend()
	resumeOffset := receivedOffset(t, statusRequest(t, secondRun, http.MethodGet, "interrupted.bin"))
	uploadChunk(t, secondRun, "interrupted.bin", resumeOffset, content[resumeOffset:])

	if resumeOffset != 1200 {
		t.Fatalf("resume offset = %d, want 1200", resumeOffset)
	}
	if !bytes.Equal(receivedFile(t, secondRun, "interrupted.bin"), content) {
		t.Fatal("resumed file differs from the original content")
	}
}

func TestUploadFailsWhenTargetIsADirectory(t *testing.T) {
	testBackend := startTestBackend(t)
	if err := os.Mkdir(filepath.Join(testBackend.config.ReceiveDir, "taken"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	response := postChunk(t, testBackend, "taken", "0", []byte("data"))

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
}
