# deck-drop
A steam deck first approach to file sharing

## Requirements

- Go 1.27 or newer (see `go.mod`)
- [golangci-lint](https://golangci-lint.run/) for `make lint` (optional)

## Desktop app

DeckDrop is a single binary: a [Wails](https://wails.io) window that embeds the backend. The backend starts and stops with the window.

Prerequisites: Go, Node.js and the Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1`). On Linux/Steam Deck also install GTK 3 and WebKitGTK 4.1 (for example `libgtk-3-dev libwebkit2gtk-4.1-dev`) and pass `-tags webkit2_41` to the commands below (`make app WAILS_FLAGS="-tags webkit2_41"`).

```sh
wails dev     # run with live reload
make app      # build the app into build/bin
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

DeckDrop works with no configuration. Settings can be overridden with environment variables or command-line flags; flags win over environment variables.

| Flag     | Environment variable   | Default          | Description                              |
| -------- | ---------------------- | ---------------- | ---------------------------------------- |
| `--port` | `DECKDROP_PORT`        | `8080`           | TCP port to listen on (1-65535)          |
| `--dir`  | `DECKDROP_RECEIVE_DIR` | `received_files` | Directory received files are saved to    |
| `--name` | `DECKDROP_DEVICE_NAME` | machine hostname | Device name shown to other devices       |

```sh
deckdrop --port 9000 --name steam-deck
DECKDROP_RECEIVE_DIR=$HOME/Downloads/DeckDrop deckdrop
```

Run `deckdrop --help` to list the options.
