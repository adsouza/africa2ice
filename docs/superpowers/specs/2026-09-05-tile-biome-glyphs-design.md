# Tile biome glyphs: a redundant shape channel at focus zoom

Date: 2026-09-05. Status: proposed design, awaiting review.

## 1. Problem

Biome identity is carried by exactly one channel: tile fill color. `climateBiomeColor`
(`pkg/render/map.go`) is deliberately an **L\* ladder** — riverine 29.4, highlands 39.0,
shrubland 51.1, savanna 59.5, desert 71.6, tundra 79.9 — spaced by lightness rather than hue
precisely because "at 7.6px chroma discrimination is weak and lightness is not," and because
hue-only separation collapses under red-green color blindness.

That ladder is a good answer to the overview-zoom problem. It is not the only answer available at
**focus zoom**, where a tile is 24 DIP rather than 7.6. At that size there is room for a second,
independent channel — shape — carrying the same distinction.

This design adds a **monochrome glyph** to each land tile at focus zoom, and to each biome swatch
in the map legend. The glyph restates what the color already says. That redundancy is the point:
it backs the L\* ladder with a channel that survives any color-vision deficiency, and it gives the
legend something to teach.

## 2. Non-goals

- **No new information.** The glyph is a function of `tile.Biome` alone. It never encodes fauna,
  resources, stress, capacity, or anything the color does not already assert. A glyph therefore
  cannot desync from the projection or mislead about game state.
- No change to `internal/domain`, `internal/application`, `gameapi.Frame`, saves, state hashes, or
  the turn contract.
- No change to input, picking, or targeting.
- No glyph at overview zoom (§5.1 measures why).
- **No glyph on band discs.** Rejected on evidence; see §10.
- No change to `climateBiomeColor`. The ladder is untouched, and the glyph is drawn over it.
- No new module dependency. `pkg/render`'s archtest allow-list
  (`internal/archtest/arch_test.go`) is unchanged — `embed` is standard library and
  `isStandardLibrary` exempts it.

## 3. Player-visible behavior

At `CameraFocus`, each **land** tile draws its biome glyph centred in the cell, in a single ink
color chosen for contrast (§6). Water, unexplored, and halo tiles draw no glyph. At
`CameraOverview` no tile draws a glyph. During the 15-tick camera transition the glyph fades in
with camera progress (§7.3).

The map legend gains the same glyph inside each biome swatch, so the vocabulary is taught rather
than guessed.

## 4. Why not color emoji

The original request was for emoji. Color emoji is **not renderable** in this stack, in any of the
three encodings a font can use. This is structural, not a gap to be worked around, and it is
recorded here so it is not re-proposed.

`text/v2` (`gotextfacesource.go`) reduces every glyph to outline segments:

```go
case font.GlyphOutline:  segs = data.Segments
case font.GlyphSVG:      segs = data.Outline.Segments   // SVG source discarded
case font.GlyphBitmap:   if data.Outline != nil { segs = data.Outline.Segments } // PNG discarded
```

`segmentsToImage` then rasterizes those segments with `image.Opaque` as the source — a
single-channel alpha mask. The internal `glyph` struct carries `scaledSegments` and `bounds` and
has no field an image could occupy.

Measured against real fonts (ebiten v2.9.10, go-text/typesetting v0.3.0):

| Font | Encoding | `GlyphData` returns | Fallback outline | Renders as |
|---|---|---|---|---|
| Apple Color Emoji | sbix (PNG) | `GlyphBitmap` 64×64, ~5.6 kB | 4 segments, `{-1 0}`/`{799 800}` — **identical for every emoji** | nothing |
| Noto Color Emoji | OpenType-SVG | `GlyphSVG` | **0 segments** | nothing |
| Noto Emoji (monochrome) | plain `glyf` | `GlyphOutline`, 66–345 segments | n/a | correctly, tintable |

Note the trap in row 1: `data.Outline != nil` reads as a real guard and **passes**, with a
plausible-looking 4 segments. Both the ebiten source and the go-text types suggest a graceful
degrade to a monochrome silhouette. Only dumping the segment values shows the fallback is
degenerate — two zero-length paths at opposite corners of the em square, carrying no information
and identical across glyphs.

## 5. Sizing

### 5.1 Why focus-only

`mapTileSize = 8`; `drawFlatTerrain` draws a 7.6 extent. `FocusScale = 3.0`, so
`geometry.Cell` is 24 DIP at focus (`camera.go:84`).

Rendering the candidate set at both sizes through ebiten's own rasterizer: at 24 px the six biome
glyphs are individually identifiable; at 7.6 px all six are indistinguishable blobs. There is no
glyph choice that rescues 7.6 px — the cell is smaller than the stroke structure of any
recognizable pictograph. Hence the hard gate on `CameraFocus`.

### 5.2 Glyph size within the cell

Glyph box is 20 DIP inside the 24 DIP cell, leaving a 2 DIP margin so adjacent tiles' glyphs do
not visually merge across the 0.4 DIP tile gap.

## 6. Ink selection

A single fixed ink fails. Measured WCAG contrast of each biome against the two candidate inks
(`#FCFCFC` and `#101010`):

| biome | L\* | light ink | dark ink | chosen |
|---|---:|---:|---:|---|
| riverine woodland | 29.4 | **9.32** | 1.99 | light |
| mountainous highlands | 39.0 | **6.53** | 2.84 | light |
| coastal shrubland | 51.1 | 4.20 | **4.42** | dark |
| savanna | 59.5 | 3.14 | **5.90** | dark |
| semi-arid desert | 71.6 | 2.13 | **8.71** | dark |
| glacial tundra | 79.9 | 1.66 | **11.15** | dark |

Fixed white bottoms out at 1.66; fixed black at 1.99. Both are unusable.

**The rule is "pick the ink with the greater contrast ratio," not a lightness threshold.** An
earlier prototype used `L* > 55 → dark`, which selects light ink for coastal shrubland (4.20) when
dark is better (4.42) — the threshold picks the worse ink for precisely the biome nearest the
coin flip. Maximizing contrast needs no tuned constant and stays optimal if a palette entry ever
changes.

Worst case under the rule is **4.42** (coastal shrubland), comfortably above WCAG 1.4.11's 3:1
requirement for non-text graphical objects. It is below the 4.5:1 text bar, which does not apply:
these are pictographs, not text, and they are redundant with the fill color rather than
load-bearing.

## 7. Architecture

### 7.1 `glyphSource`, and the escape hatch

New file `pkg/render/glyph.go`:

```go
// glyphSource yields a rasterized alpha mask for a rune at a physical pixel size.
type glyphSource interface {
    mask(r rune, physicalPx float64) *ebiten.Image
}
```

The shipping implementation is font-backed over the embedded subset. It consults a **per-rune
override map** first; an entry there supplies a hand-drawn `vector` path instead of a font glyph.

This is the hedge chosen at design time: Noto Emoji is drawn as line art, and line art is the
weakness measured in §5.1. If a specific glyph reads poorly in real play, it is replaced by adding
one override entry — no caller changes, no font swap, no re-subsetting. The override map ships
empty.

### 7.2 Rasterize at physical size, not DIP

`logicalCanvas` multiplies DIP coordinates by a device scale at draw time (`canvas.go`). A glyph
rasterized at DIP size and then scaled would be blurry on any HiDPI display. Masks are therefore
rasterized at `sizeDIP * canvas.scale` and blitted 1:1, mirroring the existing text precedent at
`map.go:780` (`Size: float64(size) * scale`).

Cache key is `(rune, physicalPx, ink)`. The map redraws every frame at 60 TPS and re-rasterizing
six outlines per frame would be pure waste. The cache is naturally bounded — 6 biomes × a small
set of sizes × 2 inks — and needs no eviction policy, but it **must** be invalidated when
`canvas.scale` changes, or a window moved between displays of different density will render stale
masks at the wrong resolution.

### 7.3 Transition

Glyph alpha follows `camera.Progress` so glyphs fade in over the existing 15-tick transition
rather than popping at a threshold.

## 8. Glyph vocabulary

| biome | glyph | note |
|---|---|---|
| Riverine woodland | 🌳 U+1F333 | |
| Savanna | 🌾 U+1F33E | busiest of the six; first override candidate |
| Coastal shrubland | 🐚 U+1F41A | |
| Mountainous highlands | ⛰ U+26F0 | |
| Semi-arid desert | 🌵 U+1F335 | **not** 🏜 U+1F3DC, which Noto draws as a framed desert scene — far too busy at 20 px |
| Glacial tundra | ❄ U+2744 | |
| Water (legend only) | 🌊 U+1F30A | legend swatch only; water tiles draw no glyph |

## 9. Legend

Current layout (`map.go:807-822`): 9 entries at `entryWidth = 96`, swatch 8×8 at `(x+4, y+4)`,
label at `x+15` size 8.5, meaning at `(x+4, y+13)` size 7, inside `mapLegendHeight = 25`.

An 8×8 swatch cannot host a legible glyph. Changes:

- Swatch grows 8×8 → 12×12, staying at `(x+4, y+4)`.
- Label x offset 15 → 19.
- Meaning text moves `y+13` → `y+14` to clear the taller swatch.
- Glyph drawn inside the swatch at 11 px with the §6 ink rule.

`escarpment` keeps its stroked-line treatment; `unknown` keeps a bare swatch. This is the most
invasive part of the change and it will touch `legend_test.go`.

## 10. Rejected alternative: glyphs on band discs

Band discs were in the original request and were **cut on measurement**. Recorded here with
evidence so the idea is not revived without new information.

At focus, `drawTileMarker` uses `radius := 3.6 * markerScale` with `markerScale = 3.0`, giving a
10.8 DIP radius — a 21.6 DIP disc whose inscribed square is ~15 DIP. Rendering candidate species
glyphs there produced two independent failures:

1. **The cue cannot carry the distinction.** At ~15 px, 🚶 and 🧍 are both an undifferentiated
   white humanoid blob. They separate cleanly at 5× magnification and not at all at 1×.
2. **The cue occludes the channel it was meant to reinforce.** On a mixed-species disc the figure
   straddles the orange/blue wedge boundary, making the species split *harder* to read than with
   no glyph at all.

The disc is also already the densest 21 px on screen, carrying species wedges plus rings at
`5.2 × markerScale` and `6.4 × markerScale`.

So the redundancy attempt is net-negative on the exact goal that motivated it, while the same idea
at 24 px tiles works. If species reinforcement is wanted, the right lever is a cue in **unused**
space — an outline ring style or a notch — not an overprinted glyph. That is a separate feature
and deliberately out of scope here.

## 11. Size, licensing, and audit

Font: **Noto Emoji**, monochrome, SIL Open Font License 1.1, from `google/fonts`
(`ofl/notoemoji/NotoEmoji[wght].ttf`). The variable font is instanced at a single weight and
subset to the §8 vocabulary, then embedded with `go:embed`.

Measured (7 glyphs, instanced at wght=400, hinting and layout tables dropped):

| | bytes |
|---|---:|
| subset raw | 6,412 |
| subset brotli `-q 11` | **4,561** |
| current release brotli | 3,288,292 |
| projected | ~3,292,853 |
| `MaxCompressedWasmBytes` | 3,600,000 |
| remaining headroom | ~307,000 |

**No §10 ratchet amendment is required.** The ceiling is untouched and the one-way ratchet is
unaffected.

Required companion changes:

- `THIRD_PARTY_NOTICES.md` gains an OFL 1.1 section reproducing the license.
- `tools/check_audits/main.go` gains an entry in its hardcoded required-section list (the file
  already asserts the bundled Go Regular font's license is present and version-identified; the new
  font needs the equivalent).

The subset is committed as a build input, so the exact bytes are reviewable and reproducible
rather than fetched at build time.

## 12. Testing

- **Golden images** per biome at true 24 DIP focus size, so regressions in glyph choice, sizing, or
  ink are caught at the size players see rather than magnified.
- **Overview gate**: assert no glyph is drawn at `CameraOverview`, and none for water, unexplored,
  or halo tiles.
- **Ink rule**: extend the `biome_contrast_test.go` pattern to assert every biome's chosen ink
  clears 3:1 against its fill, and that the chosen ink is the higher-contrast of the two. This
  test must fail if a palette entry changes such that the other ink becomes better.
- **Scale invalidation**: assert the mask cache produces a differently-sized mask after
  `canvas.scale` changes, guarding §7.2.
- **Subset budget**: assert the embedded subset stays under a committed byte ceiling, so a future
  vocabulary addition is a visible decision rather than silent growth.
- **Override hatch**: a test registering a stub override asserts it takes precedence over the font
  glyph, so the escape hatch is proven live rather than assumed.

## 13. Risks

- **Noto Emoji is line art, not silhouette.** Measured at 21 px, filled shapes survive and stroked
  contours dissolve; the six chosen glyphs are on the survivable side, but 🌾 is closest to the
  line. This is the risk the §7.1 override map exists to absorb. It is a real possibility that one
  or two glyphs end up hand-drawn, and that should not be read as the design failing.
- **Legend layout.** Moving swatch and label constants is the only change here that touches
  existing verified layout. Contained, but it is where a regression is most likely.
- **Glyph legibility is asserted from an offline rasterization** that replicates
  `segmentsToImage` (same `gvector` rasterizer, `draw.Src`, `image.Opaque`) rather than from a
  live ebiten context. Faithful, but the first implementation step should reproduce one golden
  image in-engine and confirm it matches before the rest is built on the assumption.

## 14. DESIGN.md

This feature adds a bundled asset and a player-visible presentation channel, so it warrants an
Appendix C row and a §6 presentation note. That edit is part of implementation, not of this spec.
