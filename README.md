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

## Headless backend

```sh
go build -o deckdrop ./cmd/deckdrop   # build the binary
./deckdrop                            # run it (or: go run ./cmd/deckdrop)
make test                             # run the tests
make lint                             # run the linter
```

The backend listens on port 8080, saves received files to `./received_files`, and announces itself on the local network as `_deckdrop._tcp` via mDNS so other devices can find it. Stop it with Ctrl+C.

## HTTP API

Files are sent in chunks so interrupted transfers can resume.

| Endpoint                   | Description                                                                                             |
| -------------------------- | ------------------------------------------------------------------------------------------------------- |
| `GET /health`              | Returns 200 when the backend is running                                                                 |
| `POST /upload`             | Writes the request body at a byte offset. Headers: `X-File-Name`, `X-File-Offset`. Returns 204          |
| `GET /status?filename=...` | Returns the bytes received so far in the `X-File-Offset` header, to resume from                         |

Uploads over the maximum chunk size (64 MiB) are rejected with 413. Invalid file names are rejected with 400.

```sh
curl -X POST --data-binary @photo.jpg \
  -H 'X-File-Name: photo.jpg' -H 'X-File-Offset: 0' \
  http://localhost:8080/upload
curl -i 'http://localhost:8080/status?filename=photo.jpg'
```

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
