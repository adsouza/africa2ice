# Fog halo: graded biome hint and shimmer on nearby unexplored tiles

Date: 2026-09-05. Status: approved design, awaiting implementation plan.

## 1. Problem

Unexplored tiles render in one flat opaque near-black (`unexploredTileColor`, `pkg/render/map.go`).
The boundary between the known world and the fog is a hard edge with no depth, and the map gives
no sense of a frontier that the player is standing at the edge of.

This design adds a **halo**: unexplored tiles within three rings of an explored tile show a heavily
darkened trace of what they actually are, graded by distance, with a low-amplitude shimmer that
re-rolls a few times a second. Tiles beyond three rings keep the existing flat fog.

This is a deliberate, user-approved gameplay change, not only a visual one. A three-ring biome
preview meaningfully widens effective vision: a band on a coast previews an arc of terrain it has
not explored. That was chosen with the consequence stated.

## 2. Non-goals

- No change to `internal/domain`, `internal/application`, `gameapi.Frame`, saves, state hashes, or
  the turn contract. The authoritative `Explored` bit is untouched.
- No change to input. Halo tiles stay unpickable, uninspectable, and untargetable.
- No change to markers, selection rings, passage overlays, escarpments, reachable highlights, or
  the macro-impact filter. All remain gated on the real `Explored` bit.
- No line-of-sight, no tactical fog, no decay. Exploration remains persistent and monotonic.
- No new dependency. `pkg/render`'s archtest allow-list (`internal/archtest/arch_test.go`) is
  unchanged.

## 3. Player-visible behavior

For each unexplored tile, let `d` be its Chebyshev distance to the nearest explored tile.

- `d = 1, 2, 3`: the tile is a **halo tile**. It renders as a very dark rendition of its real
  terrain — its biome if land, the current epoch water grade if sea — dimming with distance, with
  a slow per-tile shimmer.
- `d > 3`: flat `unexploredTileColor`, exactly as today.
- Explored tiles: unchanged.

Chebyshev rather than Manhattan because exploration already reveals a 3x3 footprint (DESIGN.md
"Persistent exploration fog"), so the halo grows the same shape the reveal does.

## 4. Color: a compressed L* ladder below the fog boundary

### 4.1 Why not alpha blending

The obvious implementation — `lerp(fog, terrainColor, t)` with `t` fixed per ring — is wrong, and
measurably so. `climateBiomeColor` is an **L\* ladder**: riverine 29.4, highlands 39.0, shrubland 51.1,
savanna 59.5, desert 71.6, tundra 79.9 (the palette comment names this ladder; the values here are
computed from the actual RGB entries, which are stored in a different order than the ladder reads).
Blending by a single alpha against near-black fog preserves neither the ceiling nor the spacing:

| blend `t` = 0.35 | resulting L\* |
|---|---|
| riverine | 11.7 |
| tundra | **32.3** |

Two defects. First, a fogged tundra tile at L\* 32.3 is **brighter than an explored riverine tile at
29.4** — the fog/explored distinction inverts at exactly the boundary that matters. Second, the
compression is uneven: riverine spans 8.9 L\* from fog while tundra spans 29.5, giving bright
biomes over three times the visual presence of dark ones. That is an information bias nobody chose.

Explored water is affected the same way: the humid epoch water anchor is L\* 43.4, also above
riverine.

### 4.2 The ladder

Halo color is specified in **L\* space**, then solved back to a blend. Each ring owns an L\* band
strictly between fog (L\* 2.82) and the darkest explored color (riverine, L\* 29.4). The explored
land ladder `[29.4, 79.9]` maps order-preservingly onto each ring's land band. Water gets a slot
**below all land in the same ring**.

| ring | land L\* band | water L\* | jitter amplitude |
|---|---|---|---|
| `d = 1` | 12.0 .. 24.0 | 8.0 | 1.5 L\* |
| `d = 2` | 8.0 .. 16.0 | 6.0 | 1.0 L\* |
| `d = 3` | 5.0 .. 9.5 | 4.2 | 0.6 L\* |

`L*_target(biome, d) = floor(d) + (L*(biome) - 29.4) / (79.9 - 29.4) * span(d)`

Water sits below land in every ring so that **coastline reads as a lightness edge**. At these
levels chroma discrimination has collapsed entirely; hue cannot carry the land/sea distinction.
This is the same reasoning the existing palette comment gives for the ladder at 7.6 px.

Resulting brightest halo color anywhere is L\* 24.0, against a 29.4 ceiling — 5.4 L\* of margin.

### 4.3 Solving for the blend

Along the segment `fog -> terrainColor`, every channel of every terrain color exceeds the
corresponding fog channel, so L\* is strictly monotonic in `t` and bisection is safe.

- **Land**: `haloLandBlend[biome][ring]` holds the `(tLow, tHigh)` pair achieving the ring's
  jittered floor and its target. Six biomes x three rings x two values, solved once at package
  init by bisection (~30 iterations each; microseconds).
- **Water**: the epoch water grade is continuous in `AridityIndex`, so its three ring pairs are
  solved at terrain-rebuild time and cached with the terrain key, not per tile and not per frame.

Drawn blend is `t = lerp(tLow, tHigh, n)` with `n` in `[0, 1)` from the noise. Interpolating `t`
linearly between the two solved endpoints rather than re-solving per sample introduces a
sub-0.05 L\* nonlinearity across a band at most 1.5 L\* wide — below any visible threshold.

### 4.4 Accepted ambiguity

Encoding two variables (biome and distance) in one channel means the rings overlap: a `d = 1`
riverine tile (L\* 12.0) is darker than a `d = 2` tundra tile (L\* 16.0). Distance is therefore not
readable as an absolute — it is a felt falloff, and the biome ladder is the primary read. This is
accepted, not overlooked.

## 5. Shimmer

### 5.1 Clock

Ebitengine runs at its default 60 TPS (no `SetTPS` call exists anywhere in the repo). `pkg/app`'s
`Update` increments a tick counter and hands the derived phase step to the scene via
`SetShimmerPhase`.

| constant | value | effect |
|---|---|---|
| `shimmerTickStride` | 4 | phase advances 15x/second |
| `shimmerStepsPerRoll` | 3 | each tile re-rolls its target 5x/second |

Three substeps means the easing curve is sampled at `f = 0, 1/3, 2/3`, i.e. smoothstep values
`0, 0.259, 0.741` — three distinct values per transition, so the motion is a scintillation rather
than a glide. **If that reads as too steppy, the tuning axis is `shimmerTickStride`, not the roll
count**: stride 2 gives 30 Hz stepping and six substeps at the same 5 Hz re-roll, at double the
paint rate.

### 5.2 Noise

Pure hash, no stored state, no allocation, and structurally unable to reach the simulation RNG —
`internal/archtest/arch_test.go` limits `pkg/render` to `gameapi`, `render`, `audio`, and ebiten.
This is what keeps `-screenshot` reproducible.

For a tile at `(x, y)` on phase step `s`:

```
offset = hash(x, y, 0) mod shimmerStepsPerRoll     // decorrelates tiles
cycle  = (s + offset) / shimmerStepsPerRoll        // integer division
frac   = ((s + offset) mod shimmerStepsPerRoll) / shimmerStepsPerRoll
n      = lerp(unit(hash(x, y, cycle)), unit(hash(x, y, cycle+1)), smoothstep(frac))
```

`hash` is a 32-bit integer mix (murmur3 `fmix32` over the mixed coordinates); `unit` maps it to
`[0, 1)`. The per-tile `offset` is what makes the field twinkle tile by tile instead of pulsing as
one sheet.

The jitter is **subtractive only** — it moves `t` down from the ring's target toward `tLow`. The
L\* ceiling is therefore structural rather than a clamp that a later edit could forget to apply.

Note that at `d = 1` the jitter amplitude (1.5 L\*) is comparable to the spacing between adjacent
biomes on the compressed ladder (~2.3 L\*), so shimmer can momentarily invert two neighbouring
biomes' apparent lightness. Acceptable for fog; listed in section 12 as a tuning axis.

## 6. Render integration

**Approach: recompose at 15 Hz.** The halo draws inside `drawFrame` immediately after
`drawTerrain`, and the shimmer step joins `mapFrameKey`.

This is justified by measurement, not assumption: `hover` is already a `mapFrameKey` field, so
moving the mouse across the map today rebuilds the entire composed frame every frame, and
`docs/PERFORMANCE.md` records a p95 frame gap of 16.8 ms at DPR 2 under exactly that load. A 15 Hz
recompose is roughly a quarter of a load the reference machine already sustains.

- **Fringe set** is computed by a three-ring BFS from the explored set over 6,144 tiles and cached
  with the terrain key (`TerrainRevision`, `AridityIndex`), not recomputed per shimmer step.
- **`terrainImage` is not touched.** Baking the halo into it would force a 6,144-tile rebuild
  15x/second and defeat the cache entirely.
- Halo tiles are guaranteed free of other overlays — unexplored tiles get no marker, ring, or
  reachable highlight, and a passage with neither endpoint explored is absent — so ordering the
  halo pass right after terrain raises no z-order question.

### 6.1 Named fallback

If the new performance gate (section 8) shows a problem, the fallback is **cached noise layers**:
pre-render three noise fields at terrain-revision time and blit two per frame with complementary
alpha, ~3 textured quads instead of a recompose. It is not the default because it needs a
background-restore path, costs ~4.7 MB of heap against a 24.5 MB baseline, and must suppress itself
under the notice box, end scene, and resize overlay — all of which live inside the map rectangle
and would otherwise be painted over.

## 7. Invariants and tests

1. **Lightness ceiling.** Every reachable halo color, across all six biomes, all three epoch water
   anchors, all three rings, and both jitter extremes, has L\* strictly below `climateBiomeColor`'s
   darkest entry. Property test; `pkg/render/biome_contrast_test.go` establishes the pattern.
2. **Ordering within a ring.** At fixed `d` and fixed noise, halo L\* is monotonic in the explored
   biome ladder, and water is below every land biome.
3. **Reach.** No tile at Chebyshev distance > 3 from any explored tile differs from
   `unexploredTileColor`; no explored tile is altered.
4. **Input is unchanged.** A halo tile still cannot open an inspector or become a migration target.
   This is the test that stops a visual change from quietly becoming an input change.
5. **Determinism.** Two runs at the same phase step produce byte-identical output; `-screenshot`
   remains reproducible.
6. **Repaint.** `Draw` returns true on a shimmer-step change and false on an unchanged step —
   guarding against the failure where the phase is computed but never enters the frame key, and the
   shimmer animates at 0 Hz.

## 8. Performance: the gate that currently cannot fail

`testdata/performance_profile_save.json` has **all 6,144 tiles explored**. With everything explored
there is no fringe, the halo is empty, and the shimmer never executes — so the recorded 59.88
median FPS would keep passing no matter how slow this feature is. Shipping against that gate would
be shipping unmeasured.

Two additions:

- A **Go benchmark in `pkg/render`** (where the ebiten `TestMain` lives) over a worst-case fringe,
  so a regression reports in seconds rather than via the Playwright run.
- A **partially-explored fixture** for the wasm harness, sized to maximise fringe perimeter, with
  its own recorded row in `docs/PERFORMANCE.md`.

The DPR 1 / DPR 2 floors (30 and 20 median FPS) and the 150 ms p95 frame-gap ceiling apply
unchanged to the new fixture.

## 9. Reduced motion

A settings toggle freezes the jitter at its band's target — keeping the reveal, dropping the
animation — and restores the idle-paint-nothing property for players who enable it. Nothing in
DESIGN.md currently addresses motion sensitivity, so this is new ground rather than an existing
hook.

**Implementation hazard, verified.** `ui.UISettings` carries a `SchemaVersion`, currently `2`, and
`DecodeUISettings` switches on it with explicit cases. Adding `ReducedMotion` means bumping
`UISettingsSchemaVersion` to 3 and adding a `case 3` — and `pkg/ui/ui_settings_test.go:74` asserts
that a payload with `"SchemaVersion":3` is **rejected as a future version**. That test becomes
wrong on the bump. Grep the whole repo for hardcoded schema literals compared against round-tripped
output before changing the constant; the named files are not the only ones affected.

## 10. DESIGN.md amendments

Three passages become false and must be rewritten in the same change, not afterwards:

1. **"Exploration fog and hidden-map behavior"** states the fog "needs no separate mesh,
   transparency sorting, texture asset, animation clock, fade timer, or RNG" and that no biome
   information leaks. Both clauses are revoked for the halo band; the flat-fog region beyond three
   rings retains them.
2. **The same section's** "renders every `Tile.Explored == false` cell in one fixed opaque
   near-black color" is now true only beyond `d = 3`.
3. **Section 8's idle-frame property** — "leaves the screen untouched until either the accepted
   frame or UI-local presentation key changes" — is no longer unconditional. Whenever a fringe
   exists and reduced motion is off, the screen repaints 15x/second. `docs/PERFORMANCE.md`'s
   explanation of the 59.88 FPS result needs the same qualification.

The frame projection already carries full geography for every tile
(`internal/application/snapshot.go`), so no filtering change is required; the fog has always been a
renderer-side mask, and this design keeps it there.

## 11. Rejected alternatives

- **Alpha blending with a fixed `t` per ring.** Section 4.1: breaks the L\* ceiling and biases the
  ladder. This was the original proposal and the arithmetic killed it.
- **Baking the halo into `terrainImage`.** Forces a 6,144-tile rebuild 15x/second.
- **Land/water silhouette only, or a single ring.** Considered and declined by the user in favour of
  the three-ring graded biome hint, with the vision-widening consequence stated.
- **Wall-clock phase.** Would make `-screenshot` nondeterministic. The repo counts frames
  (`ui.Toast.RemainingFrames`); this follows that idiom.

## 12. Open tuning questions

These are numbers, not structure. Each is a named constant.

1. **`shimmerTickStride = 4` (15 Hz).** Raise smoothness to 30 Hz with stride 2 if the
   scintillation reads as steppy.
2. **Jitter amplitude at `d = 1` (1.5 L\*)** against ~2.3 L\* biome spacing — may need lowering if
   the momentary inversions read as noise rather than haze.
3. **Ring L\* bands.** The 24.0 ceiling has 5.4 L\* of headroom under riverine and could be raised
   if the halo reads as too faint at real scale.

**These must be judged by looking, not by arithmetic.** A still is insufficient — `-screenshot`
writes exactly one PNG (`main.go`), and shimmer cannot be evaluated from a single frame. The
implementation plan should build a multi-frame capture (N frames at successive phase steps, as a
strip or an animation) **before** tuning any of these three, so the choice is made by observation.
