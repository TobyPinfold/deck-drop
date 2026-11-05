# Headless mode and configuration

The same binary runs without a window, for example on an always-on box that just receives files. Add `--headless`:

```sh
deckdrop --headless --pair                 # log a one-time pairing code at startup
deckdrop --headless --accept               # accept files from paired devices without asking
deckdrop --headless --port 9000 --name nas
```

A headless device has nobody to ask, so it **declines every offer unless you run it with `--accept`**, which accepts anything from paired devices.

## Options

Environment variables and flags override saved settings for that run; flags win over environment variables.

| Flag         | Environment variable   | Default                                            | Description                                                 |
| ------------ | ---------------------- | -------------------------------------------------- | ----------------------------------------------------------- |
| `--port`     | `DECKDROP_PORT`        | `8080`                                             | TCP port to listen on (1-65535)                             |
| `--dir`      | `DECKDROP_RECEIVE_DIR` | `~/Downloads/DeckDrop` (`received_files` headless) | Directory received files are saved to                       |
| `--name`     | `DECKDROP_DEVICE_NAME` | machine hostname                                   | Device name shown to other devices                          |
| `--headless` |                        | off                                                | Run without the desktop window                              |
| `--pair`     |                        | off                                                | Log a one-time pairing code at startup (headless)           |
| `--accept`   |                        | off                                                | Accept files from paired devices without asking (headless) |

```sh
DECKDROP_RECEIVE_DIR=$HOME/Downloads/DeckDrop deckdrop --headless
```

Run `deckdrop --help` to list the options.

The headless backend announces itself on the local network as `_deckdrop._tcp` via mDNS and saves received files to `./received_files` by default. Stop it with Ctrl+C.

## Trying it on one machine

Run one instance headless and one with the window, with different ports and names:

```sh
deckdrop --headless --port 8099 --name other-deck --dir /tmp/dd-test --pair --accept
deckdrop    # pair with "other-deck" using the logged code, then send it a file
```
