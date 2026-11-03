# deck-drop
A steam deck first approach to file sharing

## Requirements

- Go 1.27 or newer (see `go.mod`)
- [golangci-lint](https://golangci-lint.run/) for `make lint` (optional)

## Desktop app

DeckDrop is a single binary: a [Wails v3](https://v3.wails.io) window that embeds the backend. The backend starts and stops with the window.

Prerequisites: Go, Node.js and the Wails CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`, make sure `$(go env GOPATH)/bin` is on your `PATH`). On Linux/Steam Deck also install GTK 4 and WebKitGTK 6.0 (for example `libgtk-4-dev libwebkitgtk-6.0-dev`). Run `wails3 doctor` to check your setup.

```sh
wails3 dev    # run with live reload
make app      # build the app into bin/
```

The desktop app accepts the same flags and environment variables as the headless backend (see [Configuration](#configuration)), and its received files default to `~/Downloads/DeckDrop`. Add `--headless` to run the same binary without a window.

To try sending a file with two instances on one machine, run one headless and one with the window, using different ports and names:

```sh
bin/deckdrop --headless --port 8099 --name other-deck --dir /tmp/dd-test
bin/deckdrop    # select "other-deck", then choose or drop a file
```

## Releases

Pushing a tag such as `v0.1.0` runs the Release workflow, which builds a Linux AppImage (Steam Deck), a Windows installer and a macOS app bundle (zipped, Apple Silicon) and attaches them to a GitHub release. The macOS and Windows builds are unsigned, so macOS needs right-click → Open the first time and Windows may show a SmartScreen warning. The workflow also runs on pull requests that change `build/` so packaging problems are caught early.

## Headless backend

```sh
go build -o deckdrop ./cmd/deckdrop   # build the binary
./deckdrop                            # run it (or: go run ./cmd/deckdrop)
make test                             # run the tests
make lint                             # run the linter
```

The backend listens on port 8080, saves received files to `./received_files`, and announces itself on the local network as `_deckdrop._tcp` via mDNS so other devices can find it. Stop it with Ctrl+C.

## Go packages

The root package wires the Wails desktop application; `cmd/deckdrop` is the headless entry point. `internal/app` is the desktop-facing facade. `internal/backend` owns the receiving server lifecycle, with HTTP routes in `internal/backend/httpapi` and command-line/environment parsing in `internal/backend/config`. Discovery, sending, settings, and received-file history live in their respective `internal` packages.

## HTTP API

Files are sent in chunks so interrupted transfers can resume. The server speaks HTTPS only (TLS 1.3); see [Security](#security).

| Endpoint                   | Description                                                                                             |
| -------------------------- | ------------------------------------------------------------------------------------------------------- |
| `GET /health`              | Returns 200 when the backend is running                                                                 |
| `POST /pair`               | Pairs a device that proves it knows the one-time code (see [Security](#security))                      |
| `POST /upload`             | Writes the request body at a byte offset. Headers: `X-File-Name`, `X-File-Offset`. Returns 204          |
| `GET /status?filename=...` | Returns the bytes received so far in the `X-File-Offset` header, to resume from                         |

`/upload` and `/status` return 403 unless the caller presented the certificate of a paired device. Uploads over the maximum chunk size (64 MiB) are rejected with 413. Invalid file names are rejected with 400.

Only `/health` can be called without pairing, for example `curl -k https://localhost:8080/health` (`-k` because the certificate is self-signed).

## Security

Each device creates a self-signed certificate the first time it runs (`identity.crt` and `identity.key` in the config folder; the key is readable only by you). Its SHA-256 fingerprint identifies the device and is advertised in the mDNS `fp=` record. All transfers are encrypted with TLS 1.3.

### Pairing

Only paired devices can send files to this one. To pair, open **Pairing** on the receiving device and choose **Show pairing code**. On the other device, select the receiver, choose **Pair with ...** and enter the six digit code (the on-screen keypad works with the D-pad and A). The code expires after two minutes, works once, and is discarded after five wrong attempts. The code is never sent: the other device sends an HMAC of both certificate fingerprints keyed with the code, so someone watching the network, or sitting in the middle with their own certificate, cannot reuse it.

Pairing is one way: pairing B with A lets B send to A. Pair the other way round to send in both directions. Each side stores the other's certificate fingerprint in `paired.json` in the config folder, and the sender pins it. **Unpair** removes a device, and it can no longer send until it pairs again.

A headless device logs a one-time code at startup when run with `--pair`, for example `deckdrop --headless --pair`.

### Trust model and limits

- There is no certificate authority. A sender checks that a device's certificate matches the fingerprint it has pinned for it.
- Connections to a paired device are authenticated: the sender refuses a certificate that is not the pinned one. The fingerprint advertised over mDNS is only a hint for finding the pin; anyone can announce anything, but cannot present the paired device's certificate.
- Pairing is only as strong as the code: an attacker has five guesses at a one in a million code per pairing attempt, and the person pairing should check the code appears on the device they expect.
- Devices are still visible to everyone on the network; see issue #43.
- Deleting the identity files gives the device a new identity, which devices that pinned the old one will reject.

## Configuration

DeckDrop works with no configuration. The device name, receive folder and port can be changed in the app's Settings section; changes apply immediately and are saved to `settings.json` in your user config folder (`~/Library/Application Support/DeckDrop` on macOS, `~/.config/DeckDrop` on Linux, `%AppData%\DeckDrop` on Windows). If the new port is already in use, the previous settings stay in effect.

Environment variables and command-line flags override saved settings for that run; flags win over environment variables.

| Flag     | Environment variable   | Default          | Description                              |
| -------- | ---------------------- | ---------------- | ---------------------------------------- |
| `--port` | `DECKDROP_PORT`        | `8080`           | TCP port to listen on (1-65535)          |
| `--dir`  | `DECKDROP_RECEIVE_DIR` | `received_files` | Directory received files are saved to    |
| `--name` | `DECKDROP_DEVICE_NAME` | machine hostname | Device name shown to other devices       |
| `--headless` |                    | off              | Run without the desktop window (desktop binary only) |

```sh
deckdrop --port 9000 --name steam-deck
DECKDROP_RECEIVE_DIR=$HOME/Downloads/DeckDrop deckdrop
```

Run `deckdrop --help` to list the options.
