# Development

See also [CONTRIBUTING.md](../CONTRIBUTING.md) for commit and pull request conventions.

## Requirements

- Go 1.27 or newer (see `go.mod`)
- Node.js (current LTS) for the frontend
- The Wails CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0` (make sure `$(go env GOPATH)/bin` is on your `PATH`)
- On Linux / Steam Deck: GTK 4 and WebKitGTK 6.0 (for example `libgtk-4-dev libwebkitgtk-6.0-dev`)
- [golangci-lint](https://golangci-lint.run/) for `make lint` (optional)

Run `wails3 doctor` to check your setup.

## Common commands

```sh
wails3 dev                            # run the desktop app with live reload
make app                              # build the app into bin/
go build -o deckdrop ./cmd/deckdrop   # build the headless-only binary
make test                             # run the Go tests
make lint                             # run the linter
cd frontend && npm test               # frontend tests
```

After changing methods or types exposed to the frontend, regenerate the bindings with `wails3 generate bindings -clean=true -ts -i` and commit the result.

## Code layout

DeckDrop is a single binary: a [Wails v3](https://v3.wails.io) window that embeds the backend, which starts and stops with the window. The frontend is Preact with htm, built with Vite, in `frontend/`.

| Path                         | What it does                                                    |
| ---------------------------- | --------------------------------------------------------------- |
| `*.go` (root package)        | Wires the Wails desktop application                             |
| `cmd/deckdrop`               | Headless entry point                                            |
| `internal/app`               | Desktop-facing facade used by the frontend                      |
| `internal/backend`           | Receiving server lifecycle                                      |
| `internal/backend/httpapi`   | HTTP routes (see [Security design](security.md#http-api))       |
| `internal/backend/config`    | Command-line and environment parsing                            |
| Other `internal/` packages   | One per concern, for example `identity`, `trust`, `pairing`, `offer`, `discovery` (mDNS), `transfer` (sending), `settings` and `history` (received files) |
| `frontend/`                  | Preact UI, with gamepad and keyboard navigation                 |

## Releases

Pushing a tag such as `v0.1.0` runs the Release workflow, which builds Linux AppImages (x86_64 and arm64), a Windows installer and a macOS app bundle (zipped, Apple Silicon) and attaches them to a GitHub release. The workflow also runs on pull requests that change `build/` so packaging problems are caught early.
