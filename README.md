# Africa 2 Ice: Paleolithic Dispersal

A turn-based eco-strategy game about the dispersal of *Homo sapiens* from East Africa between 80,000 and 20,000 years before present.

The repository contains a deterministic Go simulation foundation and an Ebitengine host for desktop and WebAssembly. The current playable vertical slice renders the climate-coloured exploration map, bands, passages, campaign timeline, status panel, and Field Notes. It is not yet the complete presentation or content described by the design.

Current controls:

- click a cyan- or gold-outlined reachable tile to migrate the selected sapiens band (gold is the current recommendation);
- use the arrow keys to move a destination cursor anywhere in the band's one-turn neighborhood, including corners; Enter queues the selected migration and Esc clears it (named passages remain clickable);
- a red arrow marks the keyboard-selected or queued migration until it is cleared or the next turn resolves it;
- band rows show last-turn population and health changes, while the selected band names the leading causes of any decline;
- click any other map tile to see why it is not currently reachable, including water, uninhabitable terrain, and locked passages;
- Tab/Shift+Tab selects the next/previous sapiens band, with wraparound;
- Space ends the turn;
- 1–9 selects the named research shown in the persistent research-key legend, N splits a band, and I initiates available interbreeding;
- completing a technology triggers a breakthrough toast and updates Field Notes with context, its game effect, and a hint;
- F toggles the Field Notes panel;
- Ctrl+S (Cmd+S on macOS) quick-saves to the desktop filesystem or browser IndexedDB.

Victory, extinction, and turn-400 dispersal failure open a campaign epilogue with final
population, destination, and geographic-breadth results. Planning then stops, but Ctrl/Cmd+S can
still save the final state. Click **New Campaign** or press N to begin again with a fresh world.

The desktop build automatically resumes the newest committed Quick or Auto 1–3 save on its next
launch; manual slots remain explicit checkpoints. If the window is closed while a quick-save is
still being written, shutdown waits for that write to finish.

## Run locally

```sh
go run .
```

The desktop window initially fills up to 90% of the current monitor while preserving the game's
16:9 layout. It remains resizable and can be maximized using the normal window controls.

Desktop saves are written beneath `os.UserConfigDir()/africa2ice/saves`. Every desktop session writes a fresh JSONL diagnostic log in the system temporary directory and prints its path on startup.

## Web build

```sh
./build_web.sh --dev
python3 -m http.server --directory web 8080
```

Then open `http://localhost:8080/`. Do not open `web/index.html` through a `file://` URL: browsers
block the page from fetching the WebAssembly module in that security context, and IndexedDB also
expects an HTTP origin.

Release builds require Binaryen's `wasm-opt`:

```sh
./build_web.sh --release
```

## Verify

```sh
golangci-lint run
go vet ./...
go test ./...
go build ./...
GOOS=js GOARCH=wasm go build ./...
```

The complete product and architecture contract is in [`docs/DESIGN.md`](docs/DESIGN.md).
