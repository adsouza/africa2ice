# Africa 2 Ice: Paleolithic Dispersal

A turn-based eco-strategy game about the dispersal of *Homo sapiens* from East Africa between 80,000 and 20,000 years before present.

The repository contains the deterministic Go simulation, Ebitengine desktop/WebAssembly host,
climate-coloured exploration map, bands, passages, campaign timeline, a mouse-driven right-hand
task panel, the Field Notes drawer, filesystem/IndexedDB saves, and automated native/browser
release gates.

Current controls:

- the title scene offers Continue, New Campaign, and explicit checkpoint loading; the Game Menu's
  T shortcut returns there without changing the current campaign;
- the right-hand task panel drives the turn: band chips along the top (a +N chip opens the full
  list), the selected band's line with a details disclosure, and the THIS TURN checklist — Move,
  Research, Workforce — above the End turn button, whose label names whatever is blocking it;
- everything is clickable: chips, checklist row headers, the Move/Research/Workforce buttons, the
  workforce sliders, the drawer, and every menu and settings row. Each control has a keyboard alias
  that runs the same code, so the two paths can never disagree;
- click a cyan- or gold-outlined reachable tile to migrate the selected sapiens band (gold is the
  current recommendation); hovering any explored tile tints it and fills the Move row's TARGET
  column with that tile's liveability, while an arrow-key or queued choice keeps precedence over
  the pointer;
- use the arrow keys to move a destination cursor anywhere in the band's one-turn neighborhood,
  including corners; Enter queues the selected migration and Esc clears it (named passages remain
  clickable);
- a red arrow marks the keyboard-selected or queued migration until it is cleared or the next turn
  resolves it;
- the fixed top-down map keeps every tile aligned with its grid location; biome color, fog, reachable
  outlines, band markers, migration arrows, and ochre impassable-escarpment edges share that one
  unambiguous surface, and clicking any other tile explains why it is not currently reachable;
- band chips and the selected band's line show last-turn population and health changes, and the
  details disclosure names the leading causes of any decline;
- the Field Notes drawer sits over the lower edge of the map in three states — hidden (a full-width
  one-line bar that still carries the newest campaign event), compact, and expanded. F hides or
  shows it, Shift+F switches compact and expanded, the drawer tab and its ▲ more control do the
  same with the mouse, and the wheel scrolls longer entries, whose historical context includes
  compact references;
- Z, or the map-corner button, switches the camera between the whole-map overview and a close view
  of the selected band;
- a first-turn guide card appears in the panel on a new campaign; Next steps through it, × dismisses
  it for good, and Settings' "Show first-turn guide" brings it back;
- gameplay hotkeys: Space ends the turn; Tab/Shift+Tab select the next/previous band; Esc peels one
  layer (cursor, then popover, then the Game Menu); N splits a band; I interbreeds with a co-located
  archaic band and J cycles the highlighted partner; G cycles the heritable-trait Field Notes; B
  moves to the best tile; D toggles the band details disclosure; 1–9 choose a research target;
  M mutes; ? opens the shortcut sheet; PgUp/PgDn or Shift+Up/Shift+Down change the open checklist
  row; Ctrl+S (Cmd+S on macOS) quick-saves; F1–F3 save Manual 1–3 and Shift+F1–F3 load them;
- arrows, Enter and −/+ belong to whichever checklist row is open: Move steers and queues the
  destination cursor, Research highlights and chooses, and Workforce picks a role, steps it by one
  percentage point (Shift by five), and applies. A and D are row-owned there too — A applies an
  exact 100% draft and D discards it instead of toggling details — so both act only while Workforce
  is open; changing bands, loading, or ending the turn is blocked while a draft is dirty. W, [ and ]
  are no longer bound;
- completing a technology triggers a breakthrough toast and updates Field Notes with context, its
  game effect, and a hint;
- Esc opens the Game Menu; S opens grouped Save/Delete slots, L opens grouped Load/Delete slots,
  and O opens settings for sound and the first-turn guide (Field Notes is always reachable via F/
  Shift+F and the drawer's own controls, so the menu and Settings no longer duplicate it). Settings
  offers a pointer-driven volume slider alongside its own M mute and − / + volume keys, which act
  there only.
  The browser lists Manual 1–3, Quick Save, and rolling Auto 1–3 together.

Victory, extinction, and turn-400 dispersal failure open a campaign epilogue with final
population, destination, and geographic-breadth results. Planning then stops, but Ctrl/Cmd+S can
still save the final state. Click **New Campaign** or press N to begin again with a fresh world.

Desktop and web builds automatically resume the newest committed Quick or Auto 1–3 save on their
next launch; manual slots remain explicit checkpoints. If the window is closed while a quick-save is
still being written, shutdown waits for that write to finish.

## Run locally

On Debian or Ubuntu, install Ebitengine's native audio and graphics development headers first:

```sh
sudo apt-get install libasound2-dev libgl1-mesa-dev libx11-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

On a display-less Linux test host, also install `xvfb` and `libgl1-mesa-dri`, start an Xvfb server,
and export its `DISPLAY`; Ebitengine initializes GLFW when imported even when a test does not open a
window. The CI workflows contain the exact setup used by this repository.

```sh
go run .
```

The desktop window initially fills up to 90% of the current monitor while preserving the game's
16:9 layout. It remains resizable and can be maximized using the normal window controls.

Desktop saves are written beneath `os.UserConfigDir()/africa2ice/saves`; Field Notes visibility and
drawer height, whether the first-turn guide has been dismissed, master volume, and mute state are
stored separately in `os.UserConfigDir()/africa2ice/ui_settings.json`.
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

Release builds require Binaryen's `wasm-opt` (CI pins `version_132`; any Binaryen recent enough to
support `--enable-bulk-memory-opt` works locally, and the build script says so by name if yours is
not):

```sh
./build_web.sh --release
```

`tools/check_wasm_size.sh` measures and gates the result. It fails both when the module grows past
`MaxCompressedWasmBytes` and when that ceiling has gone stale enough to stop constraining the
build — see [`docs/PERFORMANCE.md`](docs/PERFORMANCE.md) for the current measurements.

## Cut a release

```sh
./tools/release.sh 0.0.1
```

That is the whole procedure. The script refuses before it touches the remote unless the version is
`MAJOR.MINOR.PATCH`, `HEAD` is a clean `main` identical to `origin/main`, and the tag is unused both
locally and on `origin`; every one of those mirrors a gate that `.github/workflows/release.yml`
would otherwise apply minutes later, after the tag was already published. It then pushes an
annotated tag, follows the triggered run, and prints the release URL. Pass `--no-watch` to stop
after the push.

The tag builds and publishes three unsigned portable archives — Linux amd64, Windows amd64, and
macOS arm64 — plus a `SHA256SUMS` over the exact uploaded bytes. To exercise that pipeline without
publishing anything, run the `Native release` workflow manually from the Actions tab: every job runs
except the final release-creating step. Tick `skip_verify` there to skip the verification job
too, which is worth it when the thing being debugged is the packaging itself.

## Verify

```sh
golangci-lint run
go vet ./...
go test ./...
go build ./...
GOOS=js GOARCH=wasm go build ./...
go run ./tools/check_audits
npm --prefix tools/web-e2e ci
npm --prefix tools/web-e2e run install-browser
node tools/run_wasm_go_tests.mjs
node tools/run_wasm_checkpoint.mjs
npm --prefix tools/web-e2e test
./tools/check_release_readiness.sh
./tools/release_test.sh
```

Useful display-free diagnostics are:

```sh
go run . -dumpmap
go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy reference
```

The complete product and architecture contract is in [`docs/DESIGN.md`](docs/DESIGN.md).
