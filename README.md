# Africa 2 Ice: Paleolithic Dispersal

A turn-based eco-strategy game about the dispersal of *Homo sapiens* from East Africa between 80,000 and 20,000 years before present.

The repository contains the deterministic Go simulation, Ebitengine desktop/WebAssembly host,
climate-coloured exploration map, bands, passages, campaign timeline, status panel, Field Notes,
filesystem/IndexedDB saves, and automated native/browser release gates.

Current controls:

- click a cyan- or gold-outlined reachable tile to migrate the selected sapiens band (gold is the current recommendation);
- use the arrow keys to move a destination cursor anywhere in the band's one-turn neighborhood, including corners; Enter queues the selected migration and Esc clears it (named passages remain clickable);
- a red arrow marks the keyboard-selected or queued migration until it is cleared or the next turn resolves it;
- the fixed top-down map keeps every tile aligned with its grid location; biome color, fog, reachable
  outlines, band markers, migration arrows, and ochre impassable-escarpment edges share that one
  unambiguous surface;
- band rows show last-turn population and health changes, while the selected band names the leading causes of any decline;
- click any other map tile to see why it is not currently reachable, including water, uninhabitable
  terrain, steep escarpments, and locked passages;
- Tab/Shift+Tab selects the next/previous sapiens band, with wraparound;
- Space ends the turn;
- 1–9 selects the named research shown in the persistent research-key legend, and N splits a band;
- W cycles the workforce roles, [ and ] adjust the highlighted share by one percentage point,
  A applies an exact 100% draft, and D discards it; changing bands or ending the turn is blocked while
  a draft is dirty;
- I interbreeds with a co-located archaic band; the option is offered only when one shares the
  selected band's tile, where a violet ring marks it on the map and the panel names it, and pressing
  I otherwise explains why it is unavailable;
- completing a technology triggers a breakthrough toast and updates Field Notes with context, its game effect, and a hint;
- F toggles the Field Notes panel;
- G cycles contextual Field Notes for the selected band's six heritable traits;
- M toggles mute, while - and + adjust the synthesized-effect master volume in 10% steps;
- Ctrl+S (Cmd+S on macOS) quick-saves to the desktop filesystem or browser IndexedDB.
- F1–F3 save Manual 1–3, and Shift+F1–F3 load them; loading is blocked until any dirty workforce
  draft is applied or discarded.
- P or Esc opens Pause; S opens grouped Save/Delete slots, L opens grouped Load/Delete slots,
  and O opens settings for sound and Field Notes.
  The browser lists Manual 1–3, Quick Save, and rolling Auto 1–3 together.

Victory, extinction, and turn-400 dispersal failure open a campaign epilogue with final
population, destination, and geographic-breadth results. Planning then stops, but Ctrl/Cmd+S can
still save the final state. Click **New Campaign** or press N to begin again with a fresh world.

Desktop and web builds automatically resume the newest committed Quick or Auto 1–3 save on their
next launch; manual slots remain explicit checkpoints. If the window is closed while a quick-save is
still being written, shutdown waits for that write to finish.

## Run locally

```sh
go run .
```

The desktop window initially fills up to 90% of the current monitor while preserving the game's
16:9 layout. It remains resizable and can be maximized using the normal window controls.

Desktop saves are written beneath `os.UserConfigDir()/africa2ice/saves`; Field Notes visibility,
master volume, and mute state are stored separately in `os.UserConfigDir()/africa2ice/ui_settings.json`.
Every desktop session writes a fresh JSONL diagnostic log in the system temporary directory and prints
its path on startup.

Portable release archives are unsigned. Verify the archive against the published `SHA256SUMS`;
your operating system may then require its normal one-time override for an unrecognized developer.

## Web build

```sh
./build_web.sh --dev
python3 -m http.server --directory web 8080
```

Then open `http://localhost:8080/`. Do not open `web/index.html` through a `file://` URL: browsers
block the page from fetching the WebAssembly module in that security context, and IndexedDB also
expects an HTTP origin.

Web saves and preferences stay in that site's browser storage. Operational JSONL records appear in
the browser developer console and are never uploaded by the game.

Release builds require Binaryen's `wasm-opt` (CI pins `version_131`; any Binaryen recent enough to
support `--enable-bulk-memory-opt` works locally, and the build script says so by name if yours is
not):

```sh
./build_web.sh --release
```

`tools/check_wasm_size.sh` measures and gates the result. It fails both when the module grows past
`MaxCompressedWasmBytes` and when that ceiling has gone stale enough to stop constraining the
build — see [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md) for the current measurements.

## Verify

```sh
golangci-lint run
go vet ./...
go test ./...
go build ./...
GOOS=js GOARCH=wasm go build ./...
npm --prefix tools/web-e2e ci
npm --prefix tools/web-e2e run install-browser
node tools/run_wasm_go_tests.mjs
node tools/run_wasm_checkpoint.mjs
npm --prefix tools/web-e2e test
./tools/check_release_readiness.sh
```

Useful display-free diagnostics are:

```sh
go run . -dumpmap
go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy reference
```

The complete product and architecture contract is in [`docs/DESIGN.md`](docs/DESIGN.md).
