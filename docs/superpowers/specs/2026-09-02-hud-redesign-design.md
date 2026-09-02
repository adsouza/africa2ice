# HUD redesign: task-oriented panel, mouse support, guided first turn

Date: 2026-09-02. Status: approved design, awaiting implementation plan.

## 1. Problem

The gameplay HUD packs eleven information groups into a 352 px column and a 118 px bottom strip
at font sizes down to 6.3 px. Every decision except tile selection is keyboard-only, the keys are
listed in a four-line legend, and nothing tells a new player what a turn consists of. A player
actually decides five things per band per turn: one spatial action (move, split, or interbreed),
a research target, a workforce allocation, and then whether to end the turn. None of that is
visible as a to-do.

Three pillars, in the order the user named them:

1. **Task-oriented, selectively disclosed panel.** The right column becomes a checklist for the
   selected band. Reference data lives one click away.
2. **Full mouse support** via ebitenui, with every existing hotkey preserved as an alias.
3. **Guided first turn**: a dismissible guide card that walks through the four decisions without
   ever gating input.

Retiring the bottom strip is an explicit decision (user-approved) so the map can grow to
864 × 626 logical px.

## 2. Non-goals

- No change to simulation, `gameapi.Frame`, saves, hashes, RNG, or the turn contract. Every new
  value is derived at display time or is UI-local.
- No free pan or wheel zoom of the map; the camera has exactly two states (§6).
- No new Field Notes catalog content beyond the guide card copy.
- No touch or gamepad input.
- No change to the title, end-scene, notice box, timeline rail, or terrain legend beyond moving
  the legend's reachability sentence.

## 3. Architecture

### 3.1 Packages

**New package `pkg/hud`** owns the right panel, the Field Notes drawer, and the scene overlays
(title, game menu, storage browser, settings) as ebitenui widgets. It imports
`github.com/ebitenui/ebitenui`, `pkg/gameapi`, `pkg/ui`, and `pkg/render` (palette, viewport
transform, shared geometry). It does not import the application layer.
`internal/archtest` gains a `hud` category with exactly that allow-list; `app` may import `hud`.

**`pkg/render` shrinks** to map, timeline rail, terrain legend, map overlays, camera, end scene,
resize overlay, and the notice box. Deleted: `drawHUD`, `drawTileInspector`,
`drawWorkforceDraft`, `drawResearchKeys`, `drawResearchDAG`, `drawSelectedBandInspector`,
`drawControlsReference`, `drawMenuOverlay`, `MenuOverlayRowAt`, `SettingsVolumeAt`,
`FieldNotesToggleContains`, `FieldNotesPanelContains`, `NewCampaignButtonContains`, and their
layout constants. The old code is removed, not flagged.

**Moves to `pkg/ui` as pure functions** (they interpret the frame and are needed as text):
`currentTileSummary`/`targetTileSummary` and `liveabilityLines`, `conditionForSapiensBand`,
`summarizeBandOutcome`, `sapiensBandActionLabel`, `visibleSapiensBandWindow` (retained for the
"+N" full list only), `interbreedStatus`, `campaignEraLabel`, `macroWarningLabel`.

### 3.2 Drawing order

`Game.Draw`: the map scene blits its cached presentation image as today (terrain, overlays,
camera, notice, end scene, resize overlay), then `hud.Panel.Draw(screen)` draws widgets directly
in render pixels on top. The map's frame cache keeps skipping GPU work when nothing changed; the
panel redraws each frame, which ebitenui expects.

### 3.3 DIP contract

The panel receives `render.Viewport` and `render.PresentationTransform` each frame. When
`ViewportRevision` changes it rebuilds its widget tree with positions and font sizes multiplied
by `PresentationTransform.Scale` and offset by `OffsetX/OffsetY`. ebitenui hit-tests against
`ebiten.CursorPosition()`, which is already in render pixels, the same space the widgets are laid
out in, so production needs no custom cursor source; `input.SetCursorUpdater` is used only by
tests to inject synthetic positions. This preserves DESIGN.md's rule that one logical point hits
the same control at 1×, fractional, and 2×. Text is rendered at native scale, never blitted up.

### 3.4 One-way data flow

Each `Update`, `pkg/app` builds a `hud.State` value from the accepted frame plus its UI-local
fields (selected band, migration preview, workforce draft, guide step, open row, details open,
hover tile, camera override, notice). The panel renders `State` and emits `hud.Intent` values:

```
SelectBand(id)      OpenRow(row)         ToggleDetails      ToggleNotes
MoveTo(tile)        MoveToBest           Split              Interbreed(partner)
ChooseResearch(tech)
SelectRole(role)    AdjustRole(role, deltaBP)  ApplyWorkforce   DiscardWorkforce
EndTurn(force bool)
GuideNext           GuideDismiss
CameraToggle        FocusFieldNote(kind, id)   ScrollNotes(delta)
OpenMenu / menu-scene intents: Continue, NewCampaign, OpenStorage(mode), OpenSettings,
  ReturnToTitle, SaveSlot(n), LoadSlot(n), DeleteSlot(n), SetVolume(v), ToggleMute,
  ShowGuide, Back
```

`pkg/app` maps every intent onto the same guarded method its hotkey uses today. Buttons and keys
converge on one code path, so the dirty-draft guard and End-turn blockers cannot diverge.

## 4. Panel layout (1280 × 720 logical)

Right column at x = 908, width 352, y = 68 to 700. Top to bottom:

1. **Header**: title, `☰ Menu` button; clock line `76,400 BP · Turn 12/400`; era · season · epoch
   line; macro warning line when present; `Homo sapiens N · B bands · Regions R`.
2. **Band chips**: one chip per living sapiens band in attention order. Outline gold when the
   Move predicate (§5.1) is not done, green when done. Warning tiers render as `!` (amber) and
   `!!` (red) text prefixes inside the chip so color is never the only cue. Selected chip has a
   white ring. Wraps to two lines up to eight chips; beyond that the row scrolls with the wheel
   and a `+N` chip opens the full attention-ordered list as a popover.
3. **Band line**: `Band 3 · Pop 68 (+1) · Health 100%` and a `details ▾` button. Archaic
   selection labels `Computer controlled · read only` and disables all action buttons.
4. **Details** (collapsed by default, expands in place): food last turn (needed, eaten, short,
   percent, turn); deaths last turn (five causes); heritable variants as a 2 × 3 grid, each cell
   value plus local pressure, clickable to focus that trait's Field Note; interbreeding partners
   with consequence text when present.
5. **Guide card** (§7), while not dismissed.
6. **THIS TURN** checklist: three accordion rows, exactly one open (§5.2).
7. **End turn** button with blocker label (§5.3).
8. Footer hint line: `Space ends the turn · Tab next band · ? shortcuts`.

**Field Notes drawer**: docked over the bottom of the map area (x 20–884), 94 % opaque, in one
of three states: **hidden** (tab only), **compact** (102 px, about four wrapped note lines plus
the events block), or **expanded** (300 px, about fourteen lines, so most catalog entries fit
without scrolling). The tab on its upper-right edge carries `▴ more` / `▾ less` and
`hide · F`; Shift+F toggles compact/expanded. The camera's focus window (§6) is computed against
the map area above the drawer and recenters when the drawer height changes. Below the note body a
`RECENT EVENTS` block shows the two newest events, turn-stamped and newest first; each line is
clickable and focuses that event kind's Field Note. Hidden state leaves the tab only, showing the
newest event and any breakthrough accent, as the spec already requires. Wheel
over the drawer and Shift+PgUp/PgDn scroll it. Visibility remains the persisted
`FieldNotesVisible` preference. Focus priority is unchanged.

Free vertical space (about 130 px after the guide is dismissed) goes to the open accordion row.

### 4.1 Move row, open

A HERE / TARGET comparison grid: Biome, Food, Capacity (with degradation), Water, Shelter,
Mortality (seasonal · chronic), Route (cost multiplier, turns), Archaic presence. TARGET header
names its source in the spec's existing precedence: `cursor`, then `queued`, then `hover`;
otherwise reads "hover or click an outlined tile". Relative ▲▼ marks compare TARGET to HERE.
Absolute coloring (§5.4) applies to both columns.

Buttons: `Move here · Enter` (cyan when the cursor or hovered tile is in `MigrationCandidates`,
otherwise disabled), `Best tile` (first-ranked ordinary-land candidate), `Split · N`,
`Interbreed · I` with a partner picker when more than one archaic band qualifies. Disabled
buttons show the `pkg/ui` migration diagnostic as a tooltip. Clicking an unreachable tile keeps
the existing explanatory notice.

Three visible states: idle, cursor (white border, "Enter to move"), set (green, one-line summary,
row collapses).

### 4.2 Research row, open

The nine technologies as a list: number, name, progress/cost, state (learned, current,
available, needs X). Available rows are buttons. The prerequisite graph is shown as indented
"needs" text, not as a drawn DAG.

### 4.3 Workforce row, open

Five rows: label, `−` button, draggable track, `+` button, percentage. A `›` marks the selected
role. Below: `Total` (red with "reduce N% to apply" when ≠ 100 %), worker counts, `Apply · A`
(disabled unless valid and dirty) and `Discard · D`. Editing rules, dirty-draft guard, and the
inline "Apply or discard workforce changes" message are unchanged from the spec.

## 5. State model

`hud.State` is derived every `Update` and never stored.

### 5.1 Row predicates (selected sapiens band)

| Row | Done when | Summary |
|---|---|---|
| Move | `SpatialActionUsed` ∨ `HasQueuedMigration` ∨ `HasInterbreedTarget` | queued tile biome + direction; "Split queued"; "Interbreeding with Bn" |
| Research | `HasResearchTarget` | `Firecraft 66/80 · +5.8/turn` |
| Workforce | optional; never blocks | five shares; "unapplied changes" while dirty |

Chip color uses the Move predicate, so chip and row cannot disagree.

### 5.2 Disclosure

Exactly one checklist row is open. On selection change, load, new campaign, or completed turn, the
open row defaults to the first row not done, falling back to Move. Clicking a row header,
Shift+Up/Down, or PgUp/PgDn changes it. Details and drawer are independent toggles. Open row,
details, and camera override are transient and reset with selection; drawer visibility persists.

### 5.3 End turn

Hard blocks (button disabled, label names the first): dirty workforce draft; pending arrow-key
cursor. Soft block: any sapiens band whose Move predicate is not done. The button turns amber with
`End turn · N bands still need a move`; a second click within the same turn, or Space, ends the
turn anyway. Campaign not `Ongoing` hides the button.

### 5.4 Liveability coloring (presentation-only Policy values)

| Row | Amber | Red |
|---|---|---|
| Food | stock < 1.5 × band's last-turn `RequiredFU` | stock < last-turn `RequiredFU` |
| Water | stock < 0.5 × cap | stock < 0.25 × cap |
| Capacity | degradation ≥ 0.25 | degradation ≥ 0.5 |
| Mortality | seasonal + chronic ≥ 0.004 (existing danger tier) | ≥ 0.008 |
| Shelter | natural shelter < 0.3 | — |
| Archaic | any archaic band present | — |

Values are Initial; the tiers are display interpretations and never simulation inputs.

## 6. Camera

`render.Camera{Mode, CenterTile, Progress}`, UI-local, owned by `pkg/app`.

- **Overview**: the whole 96 × 64 grid at 8 px cells from origin (20, 74), as today.
- **Focus**: 3× scale (24 px cells) centered on the selected band's tile, clamped so the visible
  window (about 36 × 26 tiles) stays inside the grid.
- Focus is requested when the Move row is open and the selected sapiens band's spatial action is
  available. It follows selection changes and returns to Overview when the row closes or the
  action is consumed. `Z` and a map-corner `Overview / Focus` button set a per-selection override.
- Transitions interpolate scale and center over 15 update ticks (250 ms at 60 TPS).
- `tilePoint` and `MapTileAt` become camera-aware; every overlay and pick goes through them. The
  1× terrain cache is drawn scaled with nearest-neighbour filtering, which is exact for flat
  cells. Hover and clicks over the drawer or panel never reach the map.

## 7. Guided first turn

A state machine in `pkg/app`: `Move → Research → Workforce → EndTurn → Closing`, plus
`Dismissed`. A step advances when its row's Done predicate flips true or on `GuideNext`;
Workforce advances on Next alone. Closing explains Tab, chip colors, and the ×. Only `GuideDismiss`
(the × in the card's corner) leaves the machine; ending turns does not. The card never blocks
input and never changes Field Notes focus priority. During the Move step the map draws a dashed
highlight around the reachable cluster.

Persistence: `GuideDismissed bool` and `FieldNotesExpanded bool` join `ui.UISettings`, schema
version 2. Version 1 payloads decode with both new fields `false`; unknown versions still fall
back to defaults. Settings gains
`Show first-turn guide`, which clears the flag and resets the machine to Move.

## 8. Input routing

Order inside `Game.Update` after storage and settings polling:

1. Scene overlays (title, menu, storage, settings) own pointer and keyboard while open.
2. `hud.Panel.Update` runs ebitenui and emits intents.
3. Map input only when ebitenui reports no widget hovered.
4. Keyboard: global keys, then row-owned keys.

**Global**: Space end turn · Tab / Shift+Tab bands · Esc · F notes · N split · I interbreed ·
J cycle partner · G cycle trait note · 1–9 research · M mute · Ctrl/Cmd+S · F1–F3, Shift+F1–F3 ·
new: `Z` camera override · `?` shortcut sheet · Shift+Up/Down and PgUp/PgDn change open row ·
Shift+PgUp/PgDn scroll the drawer · Shift+F toggles the drawer between compact and expanded.

**Row-owned** (arrows, Enter, `−`/`+`):

| Open row | Arrows | Enter | −/+ |
|---|---|---|---|
| Move | destination cursor (unchanged) | queue cursor tile | — |
| Research | Up/Down highlight | choose highlighted | — |
| Workforce | Up/Down select role; Left/Right ±1 %, Shift ±5 % | apply | ±1 % selected role |

**Removed from gameplay**: `W`, `[`, `]`, `−`/`=` volume (remain in Settings).

**Esc** peels one layer: clear cursor → close popover (partner picker, +N list, shortcut sheet) →
open menu.

**Wheel**: drawer, overflowing chip row, research list. Never the map.

**Pointer feedback**: hand cursor over clickables (ebitenui cursor management), hotkey suffix on
every button, tooltip reason on disabled buttons, filled tint plus one-line tooltip on hovered
reachable tiles.

## 9. DESIGN.md edits

- §Gameplay stats layout: rewrite around chips, checklist rows and predicates, End turn blocks,
  details contents, guide card. Keep warning tiers, attention order, Tab cycling, reachability
  overlay, queued arrow, keyboard migration, split, rejected-destination feedback.
- §Top-down terrain: drop the lower-strip sentence; add the two-state camera; the prohibition
  becomes perspective (3D camera, walls, lighting, depth), not scale. Appendix C: the Locked map
  rectangle and tile-extent rows become "map logical area (20, 74) 864 × 626" and "base cell
  8 px, drawn 7.6 px, focus scale 3×" with focus scale as Policy.
- §Field Notes: "docked over the lower edge of the map" instead of the HUD; same states.
- New Policy rows: §5.4 thresholds; drawer compact/expanded heights; `GuideDismissed`,
  `FieldNotesExpanded`, and UI settings schema 2.
- §10 dependencies: ebitenui v0.7.3, measured +169 KB brotli against a 540 KB gap.
- Architecture table: `pkg/hud`. Keyboard reference updated per §8.

## 10. Testing

- **`pkg/ui`** (no ebiten): row predicates; End turn blocker ordering; chip color; §5.4 tiers;
  guide machine transitions including advance-on-Done and persistence across turns; UI settings
  v1 → v2 decode.
- **`pkg/hud`**: `State → tree` builds for the 256-band profile frame at scale 1.0, 1.5, 2.0;
  chip overflow at 9 and 17 bands; every intent kind reachable from some widget; offscreen test
  with a synthetic cursor updater clicks End turn and a chip and asserts intents. Replaces the
  hit-rectangle tests.
- **`pkg/render`**: camera round-trips `tilePoint` ↔ `MapTileAt` in both modes and
  mid-interpolation; offscreen draw tests updated for removed HUD and the drawer.
- **`pkg/app`**: intent handlers mirror hotkey tests; mouse End turn honors the dirty-draft guard
  like Space; loaded frame with queued migration renders Move done; soft block requires a second
  click.
- **`internal/archtest`**: `hud` category; ebitenui forbidden elsewhere.
- **Web smoke test** unchanged (Move is the default open row). **WASM size gate** unchanged.
- **Manual**: `-screenshot` at turns 0 and 12 against the mockups in
  `.superpowers/brainstorm/`, on a 2× display.

## 11. Decisions log

Accordion column over list+focus and tabs · Field Notes as map drawer · chips colored by Move
predicate · vitals behind details · guide card dismissed only by × with Settings restore ·
`−`/`+` and Left/Right for workforce, Up/Down for rows · PgUp/PgDn change open row · bottom strip
retired · two-state camera with Locked rows renegotiated · soft block on End turn · ebitenui
adopted after size measurement.
