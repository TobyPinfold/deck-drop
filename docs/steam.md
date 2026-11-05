# Adding DeckDrop to Steam

Add DeckDrop to Steam to open it from Game Mode on the Steam Deck.

1. Switch to Desktop Mode and download `deckdrop-x86_64.AppImage` from the [Releases page](https://github.com/TobyPinfold/deck-drop/releases).
2. Right-click the file, open **Properties → Permissions**, and allow it to run as a program.
3. Open Steam and choose **Games → Add a Non-Steam Game to My Library**. Browse to the AppImage and add it.
4. Switch back to Game Mode. DeckDrop is in your library under **Non-Steam**.

## Library artwork

Steam shows a blank tile for non-Steam games until you give them artwork. Ready-made images are in [`build/steam`](../build/steam):

| File          | Size      | Where Steam uses it                       |
| ------------- | --------- | ----------------------------------------- |
| `capsule.png` | 600×900   | Library grid (portrait)                   |
| `wide.png`    | 920×430   | Recent games shelf and landscape grid     |
| `hero.png`    | 1920×620  | Banner at the top of the game's page      |
| `logo.png`    | 1280×720  | Logo shown over the banner (transparent)  |
| `icon.png`    | 1024×1024 | Small icon next to the name               |

In Desktop Mode, open DeckDrop's page in the Steam library. Right-click the banner to set a custom background (`hero.png`) and logo (`logo.png`). Right-click the library tile and choose **Manage → Set custom artwork** for `capsule.png` and `wide.png`. Choose the icon in the shortcut's **Properties**.
