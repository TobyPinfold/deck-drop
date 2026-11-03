.PHONY: build app test lint fmt fmt-check

# Builds the headless backend and libraries; the desktop app needs `make app`.
build:
	go build ./cmd/... ./internal/...

# Builds the desktop app into bin/ (see README for prerequisites).
app:
	wails3 build

test:
	go test ./cmd/... ./internal/...

lint:
	golangci-lint run

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files:"; gofmt -l .; exit 1)
