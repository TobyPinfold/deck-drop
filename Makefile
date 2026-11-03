.PHONY: build app test lint fmt fmt-check

# build/ holds generated Wails platform files.
GO_SOURCES := main.go cmd internal

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
	gofmt -w $(GO_SOURCES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_SOURCES))" || (echo "Unformatted files:"; gofmt -l $(GO_SOURCES); exit 1)
