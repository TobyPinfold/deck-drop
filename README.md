# deck-drop
A steam deck first approach to file sharing

## Configuration

DeckDrop works with no configuration. Settings can be overridden with environment variables or command-line flags; flags win over environment variables.

| Flag    | Environment variable   | Default          | Description                              |
| ------- | ---------------------- | ---------------- | ---------------------------------------- |
| `-port` | `DECKDROP_PORT`        | `8080`           | TCP port to listen on (1-65535)          |
| `-dir`  | `DECKDROP_RECEIVE_DIR` | `received_files` | Directory received files are saved to    |
| `-name` | `DECKDROP_DEVICE_NAME` | machine hostname | Device name shown to other devices       |

```sh
deckdrop -port 9000 -name steam-deck
DECKDROP_RECEIVE_DIR=$HOME/Downloads/DeckDrop deckdrop
```

Run `deckdrop -h` to list the options.
