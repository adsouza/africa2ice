# HUD Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the keyboard-only, eleven-panel HUD with a task-oriented accordion panel built on ebitenui, a Field Notes drawer over the map, a two-state camera, and a dismissible first-turn guide, keeping every existing hotkey as an alias.

**Architecture:** A new `pkg/hud` package renders a `hud.State` value (derived every tick by `pkg/app` from the accepted frame plus UI-local fields) as ebitenui widgets laid out in render pixels, and returns `hud.Intent` values that `pkg/app` maps onto the same guarded methods its hotkeys already call. Frame interpretation (row predicates, liveability tiers, band attention order, guide state machine) moves into `pkg/ui` as pure functions so it is testable without Ebitengine. `pkg/render` keeps the map, overlays, camera, timeline, legend, notice, and end scene, and loses every HUD drawing function.

**Tech Stack:** Go 1.26.4, Ebitengine v2.9.10 (`text/v2`, `vector`), ebitenui v0.7.3, golangci-lint v2.12.2 (depguard strict allow-lists), Playwright smoke test (unchanged).

**Spec:** `docs/superpowers/specs/2026-09-02-hud-redesign-design.md` — read it first; section numbers below (§N) refer to it.

## Global Constraints

- Logical presentation is fixed at `1280 × 720` DIPs; every layout constant is authored in DIPs and multiplied by `render.PresentationTransform.Scale` at build time. No handler compares render pixels with a logical rectangle (DESIGN.md §"High-DPI viewport").
- Nothing new enters `gameapi.Frame`, `World`, saves, hashes, or RNG. All panel state is derived per tick or UI-local.
- Import boundaries are enforced twice: `.golangci.yml` depguard strict allow-lists and `internal/archtest`. Both must be edited together. `pkg/hud` may import: stdlib, `pkg/gameapi`, `pkg/ui`, `pkg/render`, `github.com/hajimehoshi/ebiten/v2`, `github.com/ebitenui/ebitenui`, `golang.org/x/image`. `pkg/app` additionally gains `github.com/ebitenui/ebitenui` (for `input.UIHovered`) and `pkg/hud`.
- Every module in `go list -m all` must have a row in `THIRD_PARTY_NOTICES.md`; `go run ./tools/check_audits` enforces it.
- WASM release size gate: `tools/wasm_size_budget.env` `MaxCompressedWasmBytes=3600000`; measured ebitenui cost is +169 KB brotli. Do not edit the budget file.
- Text is rendered at native scale, never blitted up.
- Keep all current hotkeys working except: `W`, `[`, `]` removed; `-`/`=` volume removed from gameplay (kept in Settings).
- The web smoke test (`tools/web-e2e/smoke.mjs`) presses, in gameplay: `Escape`, arrows, `Enter`, `Space`, `Control+S`, `F1`, `Shift+F1`. It must keep passing unchanged. It relies on Esc toggling the menu when no cursor is pending and on Move being the open row after a new campaign.
- Verification commands (run from repo root): `go build ./... && go vet ./... && go test ./... && golangci-lint run`. Fast single-package loops: `go test ./pkg/ui/ -run <Name> -v`.
- Commit after every task with the trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`. Work on branch `hud-redesign`.

## File structure

**New**
- `pkg/ui/bands.go` (+ `_test.go`): attention order, condition tiers, chip progress. Moved from `pkg/render/band_window.go`.
- `pkg/ui/checklist.go` (+ `_test.go`): row predicates, summaries, default open row, End-turn gate.
- `pkg/ui/liveability.go` (+ `_test.go`): tile liveability summaries and HERE/TARGET comparison rows with tiers. Moved from `pkg/render/tile_info.go`.
- `pkg/ui/guide.go` (+ `_test.go`): first-turn guide state machine.
- `pkg/hud/doc.go`, `state.go`, `intent.go`, `theme.go`, `fixed_layout.go`, `panel.go`, `header.go`, `chips.go`, `details.go`, `move_row.go`, `research_row.go`, `workforce_row.go`, `end_turn.go`, `drawer.go`, `guide_card.go`, `overlays.go`, `end_scene.go` (+ tests `panel_test.go`, `fixed_layout_test.go`, `overlays_test.go`).
- `pkg/render/camera.go` (+ `_test.go`).
- `pkg/app/hud_state.go`, `pkg/app/hud_intents.go`, `pkg/app/input.go` (+ tests).

**Modified**
- `go.mod`, `go.sum`, `.golangci.yml`, `THIRD_PARTY_NOTICES.md`, `internal/archtest/arch_test.go`.
- `pkg/ui/ui_settings.go` (+ test): schema 2.
- `pkg/render/map.go`: delete HUD drawing, camera-aware `tilePoint`/`MapTileAt`, new `Draw` signature. `pkg/render/map_test.go` accordingly.
- `pkg/app/game.go`, `pkg/app/game_test.go`.
- `docs/DESIGN.md`.

**Deleted**
- `pkg/render/band_window.go`, `pkg/render/band_window_test.go`, `pkg/render/tile_info.go`, `pkg/render/tile_info_test.go`, `pkg/render/outcome.go`, `pkg/render/outcome_test.go` (after their content moves to `pkg/ui`).

Phases: **1** pure logic and dependency plumbing (Tasks 1–6) · **2** panel, HUD removal, input routing (Tasks 7–13) · **3** camera (Tasks 14–15) · **4** guide, settings, DESIGN.md, final verification (Tasks 16–18). Each phase ends green.

---

## Phase 1 — Pure logic and dependency plumbing

### Task 1: Add the ebitenui dependency and open the import boundaries

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `.golangci.yml` (rules `render-adapter-dependencies` area, add `hud-adapter-dependencies`; extend `ebitengine-host-is-the-composition-root`)
- Modify: `internal/archtest/arch_test.go:85-140`
- Modify: `THIRD_PARTY_NOTICES.md`
- Create: `pkg/hud/doc.go`

**Interfaces:**
- Produces: the module `github.com/ebitenui/ebitenui v0.7.3` importable from `pkg/hud` and `pkg/app`; a `hud` archtest category.

- [ ] **Step 1: Add the module**

```bash
go get github.com/ebitenui/ebitenui@v0.7.3 && go mod tidy
```

Expected: `go.mod` gains `github.com/ebitenui/ebitenui v0.7.3` (it will sit under `// indirect` until Step 2 imports it), plus indirect `github.com/frustra/bbcode`, `golang.org/x/exp`, `golang.org/x/exp/shiny`, `golang.design/x/clipboard`, `golang.org/x/mobile`, and test-only modules from ebitenui's own go.mod (`github.com/matryer/is`, `github.com/stretchr/testify`, `github.com/stretchr/objx`, `github.com/davecgh/go-spew`, `github.com/pmezard/go-difflib`, `github.com/kr/pretty`, `github.com/google/go-cmp`, `gopkg.in/check.v1`, `gopkg.in/yaml.v3`). Existing pins `purego v0.9.1` and `xgb v1.2.0` stay (MVS picks the higher).

- [ ] **Step 2: Create the package so the import is real**

Create `pkg/hud/doc.go`:

```go
// Package hud is the ebitenui-based gameplay chrome: the task-oriented right
// panel, the Field Notes drawer, the first-turn guide card, and the scene
// overlays. It renders a State value the application derives every tick and
// returns Intent values; it never touches the simulation port, storage, or
// saves, and every layout constant it owns is authored in DIPs and scaled at
// build time.
package hud

import _ "github.com/ebitenui/ebitenui"
```

Run: `go build ./...` — Expected: succeeds and `go mod tidy` now lists ebitenui as a direct requirement.

- [ ] **Step 3: Write the failing archtest expectation**

In `internal/archtest/arch_test.go`, add to the table test (the function containing `{name: "render adapter", imported: module + "/pkg/render", allowed: false}`) a new test function:

```go
func TestHUDCategoryAllowsEbitenUIAndNothingElseDoes(t *testing.T) {
	ebitenui := "github.com/ebitenui/ebitenui/widget"
	if violation := importViolation("pkg/hud/panel.go", ebitenui); violation != "" {
		t.Fatalf("hud may import ebitenui: %s", violation)
	}
	if violation := importViolation("pkg/hud/panel.go", module+"/pkg/ui"); violation != "" {
		t.Fatalf("hud may import ui: %s", violation)
	}
	if violation := importViolation("pkg/app/game.go", "github.com/ebitenui/ebitenui/input"); violation != "" {
		t.Fatalf("app may read ebitenui input state: %s", violation)
	}
	for _, file := range []string{"pkg/render/map.go", "pkg/ui/bands.go", "internal/application/service.go"} {
		if importViolation(file, ebitenui) == "" {
			t.Fatalf("%s accepted an ebitenui import", file)
		}
	}
	if importViolation("pkg/hud/panel.go", module+"/internal/application") == "" {
		t.Fatal("hud accepted an application import")
	}
}
```

Run: `go test ./internal/archtest/ -run TestHUDCategory -v` — Expected: FAIL. Today `pkg/hud/` falls into the `other` category, which has no strict allow-list, so the final assertion fails: an `internal/application` import from hud is accepted. The `pkg/app` assertion also fails because ebitenui is not in the app allow-list.

- [ ] **Step 4: Add the category**

In `packageCategory`, before `case strings.HasPrefix(file, "pkg/app/")`:

```go
	case strings.HasPrefix(file, "pkg/hud/"):
		return "hud"
```

In `allowedImports`, add before `case "app":`

```go
	case "hud":
		return []string{module + "/pkg/gameapi", module + "/pkg/ui", module + "/pkg/render", "github.com/hajimehoshi/ebiten/v2", "github.com/ebitenui/ebitenui", "golang.org/x/image"}, true
```

and change the `app` case to:

```go
	case "app":
		return []string{module + "/internal/application", module + "/internal/adapters/logging", module + "/internal/adapters/storage", module + "/pkg/gameapi", module + "/pkg/render", module + "/pkg/ui", module + "/pkg/hud", module + "/pkg/audio", "github.com/hajimehoshi/ebiten/v2", "github.com/ebitenui/ebitenui"}, true
```

Run: `go test ./internal/archtest/ -v` — Expected: PASS.

- [ ] **Step 5: Mirror the rules in `.golangci.yml`**

After the `render-adapter-dependencies` rule add:

```yaml
        hud-adapter-dependencies:
          list-mode: strict
          files: ["**/pkg/hud/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/pkg/gameapi"
            - "github.com/adsouza/africa2ice/pkg/ui"
            - "github.com/adsouza/africa2ice/pkg/render"
            - "github.com/hajimehoshi/ebiten/v2"
            - "github.com/ebitenui/ebitenui"
            - "golang.org/x/image"
```

In `ebitengine-host-is-the-composition-root` `allow`, append:

```yaml
            - "github.com/adsouza/africa2ice/pkg/hud"
            - "github.com/ebitenui/ebitenui"
```

Run: `golangci-lint config verify && golangci-lint run` — Expected: no findings.

- [ ] **Step 6: Record the new modules in THIRD_PARTY_NOTICES.md**

Insert these rows into the `## Go module build list` table, keeping the table sorted by module path (rows shown with the exact license each module ships):

```markdown
| Go | `github.com/davecgh/go-spew` | `v1.1.1` | ISC | Complete Go build list only | [source](https://pkg.go.dev/github.com/davecgh/go-spew@v1.1.1?tab=licenses) |
| Go | `github.com/ebitenui/ebitenui` | `v0.7.3` | MIT | Release import graph | [source](https://pkg.go.dev/github.com/ebitenui/ebitenui@v0.7.3?tab=licenses) |
| Go | `github.com/frustra/bbcode` | `v0.0.0-20201127003707-6ef347fbe1c8` | MIT | Release import graph | [source](https://github.com/frustra/bbcode/blob/6ef347fbe1c8/LICENSE) |
| Go | `github.com/google/go-cmp` | `v0.6.0` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/github.com/google/go-cmp@v0.6.0?tab=licenses) |
| Go | `github.com/kr/pretty` | `v0.3.1` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/kr/pretty@v0.3.1?tab=licenses) |
| Go | `github.com/matryer/is` | `v1.4.1` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/matryer/is@v1.4.1?tab=licenses) |
| Go | `github.com/pmezard/go-difflib` | `v1.0.0` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/github.com/pmezard/go-difflib@v1.0.0?tab=licenses) |
| Go | `github.com/stretchr/objx` | `v0.5.2` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/stretchr/objx@v0.5.2?tab=licenses) |
| Go | `github.com/stretchr/testify` | `v1.10.0` | MIT | Complete Go build list only | [source](https://pkg.go.dev/github.com/stretchr/testify@v1.10.0?tab=licenses) |
| Go | `golang.design/x/clipboard` | `v0.7.0` | MIT | Complete Go build list only | [source](https://pkg.go.dev/golang.design/x/clipboard@v0.7.0?tab=licenses) |
| Go | `golang.org/x/exp` | `v0.0.0-20250305212735-054e65f0b394` | BSD-3-Clause | Release import graph | [source](https://pkg.go.dev/golang.org/x/exp@v0.0.0-20250305212735-054e65f0b394?tab=licenses) |
| Go | `golang.org/x/exp/shiny` | `v0.0.0-20250305212735-054e65f0b394` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/exp/shiny@v0.0.0-20250305212735-054e65f0b394?tab=licenses) |
| Go | `golang.org/x/mobile` | `v0.0.0-20231127183840-76ac6878050a` | BSD-3-Clause | Complete Go build list only | [source](https://pkg.go.dev/golang.org/x/mobile@v0.0.0-20231127183840-76ac6878050a?tab=licenses) |
| Go | `gopkg.in/check.v1` | `v1.0.0-20201130134442-10cb98267c6c` | BSD-2-Clause | Complete Go build list only | [source](https://pkg.go.dev/gopkg.in/check.v1@v1.0.0-20201130134442-10cb98267c6c?tab=licenses) |
| Go | `gopkg.in/yaml.v3` | `v3.0.1` | MIT | Complete Go build list only | [source](https://pkg.go.dev/gopkg.in/yaml.v3@v3.0.1?tab=licenses) |
```

The three "Release import graph" modules need license texts under `## License texts for release-imported modules`. Add a `### MIT License` subsection stating it applies to `github.com/ebitenui/ebitenui` (Copyright 2020 Maik Schreiber) and `github.com/frustra/bbcode` (Copyright (C) 2015 Frustra) followed by the standard MIT text in a ```text fence, and extend the existing BSD-3-Clause subsection's list of modules with `golang.org/x/exp` (copy the LICENSE text from `$(go env GOMODCACHE)/golang.org/x/exp@v0.0.0-20250305212735-054e65f0b394/LICENSE` if the existing subsection's text differs).

Run: `go run ./tools/check_audits` — Expected: exit 0. If it names a missing or mismatched module, compare with `go list -m all` and fix the row.

- [ ] **Step 7: Full check and commit**

```bash
go build ./... && go vet ./... && go test ./... && golangci-lint run
git add go.mod go.sum .golangci.yml internal/archtest/arch_test.go THIRD_PARTY_NOTICES.md pkg/hud/doc.go
git commit -m "Add ebitenui and open the hud import boundary

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 2: Move band attention order and warning tiers into pkg/ui

**Files:**
- Create: `pkg/ui/bands.go`, `pkg/ui/bands_test.go`
- Modify: `pkg/app/game.go` (callers of `render.SapiensBandIDsByAttention` at `ensureSelection` and `selectSapiens`)
- Leave `pkg/render/band_window.go` in place until Task 11 deletes it with the HUD.

**Interfaces:**
- Produces:
  - `type SapiensBandCondition uint8` with `BandStable`, `BandDanger`, `BandSuffering`; `func (c SapiensBandCondition) Marker() string` returning `""`, `"!"`, `"!!"`.
  - `func ConditionForSapiensBand(band gameapi.Band) SapiensBandCondition`
  - `func SapiensBandIDsByAttention(bands []gameapi.Band) []gameapi.BandID`
  - `func SapiensBandOrdinal(bands []gameapi.Band, id gameapi.BandID) (int, bool)`
  - `func MoveDone(band gameapi.Band) bool` — `SpatialActionUsed || HasQueuedMigration || HasInterbreedTarget`
  - `func BandsNeedingMove(bands []gameapi.Band) int`

- [ ] **Step 1: Write the failing tests**

Create `pkg/ui/bands_test.go`:

```go
package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestConditionForSapiensBandUsesSpecTiers(t *testing.T) {
	tests := []struct {
		name string
		band gameapi.Band
		want SapiensBandCondition
	}{
		{name: "healthy", band: gameapi.Band{Health: 0.95}, want: BandStable},
		{name: "health below 0.80 is danger", band: gameapi.Band{Health: 0.79}, want: BandDanger},
		{name: "mortality at 0.004 is danger", band: gameapi.Band{Health: 1, SeasonalMortalityRate: 0.002, ChronicMortalityRate: 0.002}, want: BandDanger},
		{name: "health below 0.50 is suffering", band: gameapi.Band{Health: 0.49}, want: BandSuffering},
		{name: "food shortfall last turn is suffering", band: gameapi.Band{Health: 1, LastFoodReport: gameapi.FoodTurnReport{Turn: 3, RequiredFU: 10, DeficitFU: 1}}, want: BandSuffering},
		{name: "population decline last turn is suffering", band: gameapi.Band{Health: 1, LastOutcomeReport: gameapi.OutcomeReport{Turn: 3, StartingPopulation: 100, EndingPopulation: 99, StartingHealth: 1, EndingHealth: 1}}, want: BandSuffering},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ConditionForSapiensBand(test.band); got != test.want {
				t.Fatalf("condition = %d, want %d", got, test.want)
			}
		})
	}
	if BandStable.Marker() != "" || BandDanger.Marker() != "!" || BandSuffering.Marker() != "!!" {
		t.Fatal("condition markers do not match the spec's chip prefixes")
	}
}

func TestSapiensBandIDsByAttentionOrdersSufferingFirstThenHealthThenID(t *testing.T) {
	bands := []gameapi.Band{
		{ID: 1, Species: gameapi.HomoSapiens, Population: 10, Health: 0.9},
		{ID: 2, Species: gameapi.ArchaicHominin, Population: 10, Health: 0.1},
		{ID: 3, Species: gameapi.HomoSapiens, Population: 10, Health: 0.4},
		{ID: 4, Species: gameapi.HomoSapiens, Population: 0, Health: 0.1},
		{ID: 5, Species: gameapi.HomoSapiens, Population: 10, Health: 0.9},
		{ID: 6, Species: gameapi.HomoSapiens, Population: 10, Health: 0.7},
	}
	got := SapiensBandIDsByAttention(bands)
	want := []gameapi.BandID{3, 6, 1, 5}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
	if ordinal, ok := SapiensBandOrdinal(bands, 5); !ok || ordinal != 3 {
		t.Fatalf("ordinal of band 5 = (%d, %t), want (3, true)", ordinal, ok)
	}
	if _, ok := SapiensBandOrdinal(bands, 2); ok {
		t.Fatal("archaic band received an attention ordinal")
	}
}

func TestMoveDoneAndBandsNeedingMove(t *testing.T) {
	if MoveDone(gameapi.Band{}) {
		t.Fatal("fresh band reported as done")
	}
	for _, band := range []gameapi.Band{{SpatialActionUsed: true}, {HasQueuedMigration: true}, {HasInterbreedTarget: true}} {
		if !MoveDone(band) {
			t.Fatalf("band %+v not reported as done", band)
		}
	}
	bands := []gameapi.Band{
		{ID: 1, Species: gameapi.HomoSapiens, Population: 5},
		{ID: 2, Species: gameapi.HomoSapiens, Population: 5, HasQueuedMigration: true},
		{ID: 3, Species: gameapi.ArchaicHominin, Population: 5},
		{ID: 4, Species: gameapi.HomoSapiens, Population: 0},
	}
	if got := BandsNeedingMove(bands); got != 1 {
		t.Fatalf("BandsNeedingMove = %d, want 1", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui/ -run 'TestConditionForSapiensBand|TestSapiensBandIDsByAttention|TestMoveDone' -v` — Expected: FAIL, undefined symbols.

- [ ] **Step 3: Implement `pkg/ui/bands.go`**

Port the logic from `pkg/render/band_window.go` verbatim, exported:

```go
package ui

import (
	"sort"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// Presentation-only warning tiers (DESIGN.md "Sapiens band warning thresholds").
// The simulation never reads these.
const (
	bandSufferingHealthThreshold = 0.50
	bandDangerHealthThreshold    = 0.80
	bandDangerMortalityRate      = 0.004
)

type SapiensBandCondition uint8

const (
	BandStable SapiensBandCondition = iota
	BandDanger
	BandSuffering
)

// Marker is the text prefix a chip carries so the tier is never color-only.
func (condition SapiensBandCondition) Marker() string {
	switch condition {
	case BandDanger:
		return "!"
	case BandSuffering:
		return "!!"
	default:
		return ""
	}
}

func ConditionForSapiensBand(band gameapi.Band) SapiensBandCondition {
	outcome := band.LastOutcomeReport
	declinedLastTurn := outcome.Turn != 0 &&
		(outcome.EndingPopulation < outcome.StartingPopulation || outcome.EndingHealth < outcome.StartingHealth-1e-9)
	shortOfFoodLastTurn := band.LastFoodReport.Turn != 0 && band.LastFoodReport.DeficitFU > 0
	if declinedLastTurn || shortOfFoodLastTurn || band.Health < bandSufferingHealthThreshold {
		return BandSuffering
	}
	if band.Health < bandDangerHealthThreshold || band.SeasonalMortalityRate+band.ChronicMortalityRate >= bandDangerMortalityRate {
		return BandDanger
	}
	return BandStable
}

// MoveDone is the checklist predicate for the spatial action (spec §5.1). Chips
// and the Move row both read it, so they cannot disagree.
func MoveDone(band gameapi.Band) bool {
	return band.SpatialActionUsed || band.HasQueuedMigration || band.HasInterbreedTarget
}

// BandsNeedingMove counts living sapiens bands whose spatial action is open.
func BandsNeedingMove(bands []gameapi.Band) int {
	count := 0
	for _, band := range bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 && !MoveDone(band) {
			count++
		}
	}
	return count
}

// SapiensBandIDsByAttention returns living player bands in the presentation-
// only attention order: suffering before danger before stable, then lower
// health, larger last-turn food deficit, higher projected mortality, larger
// proportional loss, ascending ID.
func SapiensBandIDsByAttention(bands []gameapi.Band) []gameapi.BandID {
	indices := sapiensBandIndicesByAttention(bands)
	ids := make([]gameapi.BandID, len(indices))
	for ordinal, index := range indices {
		ids[ordinal] = bands[index].ID
	}
	return ids
}

// SapiensBandOrdinal reports a band's zero-based position in attention order.
func SapiensBandOrdinal(bands []gameapi.Band, id gameapi.BandID) (int, bool) {
	for ordinal, bandID := range SapiensBandIDsByAttention(bands) {
		if bandID == id {
			return ordinal, true
		}
	}
	return 0, false
}

func sapiensBandIndicesByAttention(bands []gameapi.Band) []int {
	indices := make([]int, 0, len(bands))
	for index, band := range bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 {
			indices = append(indices, index)
		}
	}
	sort.Slice(indices, func(left, right int) bool {
		return sapiensBandNeedsMoreAttention(bands[indices[left]], bands[indices[right]])
	})
	return indices
}

func sapiensBandNeedsMoreAttention(left, right gameapi.Band) bool {
	leftCondition, rightCondition := ConditionForSapiensBand(left), ConditionForSapiensBand(right)
	if leftCondition != rightCondition {
		return leftCondition > rightCondition
	}
	if left.Health != right.Health {
		return left.Health < right.Health
	}
	leftDeficit, rightDeficit := left.LastFoodReport.DeficitFraction(), right.LastFoodReport.DeficitFraction()
	if leftDeficit != rightDeficit {
		return leftDeficit > rightDeficit
	}
	leftMortality := left.SeasonalMortalityRate + left.ChronicMortalityRate
	rightMortality := right.SeasonalMortalityRate + right.ChronicMortalityRate
	if leftMortality != rightMortality {
		return leftMortality > rightMortality
	}
	leftLoss, rightLoss := lastTurnPopulationLossFraction(left), lastTurnPopulationLossFraction(right)
	if leftLoss != rightLoss {
		return leftLoss > rightLoss
	}
	return left.ID < right.ID
}

func lastTurnPopulationLossFraction(band gameapi.Band) float64 {
	report := band.LastOutcomeReport
	if report.Turn == 0 || report.StartingPopulation == 0 || report.EndingPopulation >= report.StartingPopulation {
		return 0
	}
	return float64(report.StartingPopulation-report.EndingPopulation) / float64(report.StartingPopulation)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/ui/ -v` — Expected: PASS.

- [ ] **Step 5: Switch the app to the ui copy**

In `pkg/app/game.go`, replace both occurrences of `render.SapiensBandIDsByAttention(g.frame.Bands)` (in `ensureSelection` and `selectSapiens`) with `ui.SapiensBandIDsByAttention(g.frame.Bands)`.

Run: `go build ./... && go test ./pkg/app/` — Expected: PASS (`TestBandSelectionUsesAttentionOrder` still passes because the order is identical).

- [ ] **Step 6: Commit**

```bash
git add pkg/ui/bands.go pkg/ui/bands_test.go pkg/app/game.go
git commit -m "Move band attention order and warning tiers into pkg/ui

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 3: Checklist row predicates, summaries, and the End-turn gate

**Files:**
- Create: `pkg/ui/checklist.go`, `pkg/ui/checklist_test.go`

**Interfaces:**
- Produces:
  - `type ChecklistRow uint8` with `RowMove`, `RowResearch`, `RowWorkforce`, `ChecklistRowCount`; `func (r ChecklistRow) Title() string` ("Move", "Research", "Workforce").
  - `func ResearchDone(band gameapi.Band) bool`
  - `func DefaultOpenRow(band *gameapi.Band) ChecklistRow` — first not-done row, else `RowMove`.
  - `func MoveSummary(frame *gameapi.Frame, band gameapi.Band) string`
  - `func ResearchSummary(band gameapi.Band) string`
  - `func WorkforceSummary(allocation [gameapi.AssignmentCount]uint16, dirty bool) string`
  - `type EndTurnGate struct { Enabled, Soft bool; Label string }`
  - `func EndTurnGateFor(frame *gameapi.Frame, dirtyDraft, pendingCursor, armed bool) EndTurnGate`
  - `func RoleShortLabel(role gameapi.WorkforceRole) string` ("Foraging", "Hunt / fish", "Toolcraft", "Megafauna", "Shelter / care").
  - `func CompassDirection(frame *gameapi.Frame, from, to gameapi.TileID) string` ("N", "NE", … or "" when not adjacent).

- [ ] **Step 1: Write the failing tests**

Create `pkg/ui/checklist_test.go`:

```go
package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func checklistFrame() *gameapi.Frame {
	return &gameapi.Frame{
		CampaignResult: gameapi.Ongoing,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 5, Y: 5, Land: true, Explored: true, Biome: gameapi.Savanna},
			{ID: 1, X: 6, Y: 4, Land: true, Explored: true, Biome: gameapi.RiverineWoodland},
		},
		Bands: []gameapi.Band{
			{ID: 7, Species: gameapi.HomoSapiens, Population: 60, TileID: 0},
			{ID: 8, Species: gameapi.HomoSapiens, Population: 60, TileID: 0},
		},
	}
}

func TestDefaultOpenRowIsFirstUnfinishedRow(t *testing.T) {
	band := gameapi.Band{Species: gameapi.HomoSapiens}
	if got := DefaultOpenRow(&band); got != RowMove {
		t.Fatalf("fresh band opens %v, want Move", got)
	}
	band.HasQueuedMigration = true
	if got := DefaultOpenRow(&band); got != RowResearch {
		t.Fatalf("moved band opens %v, want Research", got)
	}
	band.HasResearchTarget = true
	if got := DefaultOpenRow(&band); got != RowMove {
		t.Fatalf("finished band opens %v, want Move fallback", got)
	}
	if got := DefaultOpenRow(nil); got != RowMove {
		t.Fatalf("nil band opens %v, want Move", got)
	}
}

func TestSummariesNameTheAcceptedState(t *testing.T) {
	frame := checklistFrame()
	band := frame.Bands[0]
	if got := MoveSummary(frame, band); got != "Choose a destination" {
		t.Fatalf("idle move summary = %q", got)
	}
	band.HasQueuedMigration, band.QueuedMigration = true, 1
	if got := MoveSummary(frame, band); got != "Move set → Riverine Woodland, NE" {
		t.Fatalf("queued move summary = %q", got)
	}
	band = frame.Bands[0]
	band.HasInterbreedTarget, band.InterbreedTargetID = true, 12
	if got := MoveSummary(frame, band); got != "Interbreeding with B12" {
		t.Fatalf("interbreed summary = %q", got)
	}
	band = frame.Bands[0]
	band.SpatialActionUsed = true
	if got := MoveSummary(frame, band); got != "Split queued" {
		t.Fatalf("split summary = %q", got)
	}

	research := gameapi.Band{}
	if got := ResearchSummary(research); got != "No target · choose one" {
		t.Fatalf("idle research summary = %q", got)
	}
	research.HasResearchTarget, research.ResearchTarget = true, gameapi.Firecraft
	research.ResearchProgress[gameapi.Firecraft] = 66
	research.ResearchOptions[gameapi.Firecraft].Cost = 80
	research.OriginalResearchGainPreview = 5.8
	if got := ResearchSummary(research); got != "Firecraft 66/80 · +5.8/turn" {
		t.Fatalf("research summary = %q", got)
	}

	allocation := [gameapi.AssignmentCount]uint16{3_500, 3_000, 1_500, 500, 1_500}
	if got := WorkforceSummary(allocation, false); got != "Forage 35 · Hunt 30 · Tools 15 · Mega 5 · Shelter 15" {
		t.Fatalf("workforce summary = %q", got)
	}
	if got := WorkforceSummary(allocation, true); got != "Unapplied changes" {
		t.Fatalf("dirty workforce summary = %q", got)
	}
}

func TestEndTurnGateOrdersHardBlocksBeforeTheSoftBlock(t *testing.T) {
	frame := checklistFrame()
	if gate := EndTurnGateFor(frame, true, true, false); gate.Enabled || gate.Label != "End turn · apply or discard workforce changes" {
		t.Fatalf("dirty draft gate = %+v", gate)
	}
	if gate := EndTurnGateFor(frame, false, true, false); gate.Enabled || gate.Label != "End turn · queue or clear the arrow-key choice" {
		t.Fatalf("pending cursor gate = %+v", gate)
	}
	soft := EndTurnGateFor(frame, false, false, false)
	if !soft.Enabled || !soft.Soft || soft.Label != "End turn · 2 bands still need a move" {
		t.Fatalf("soft gate = %+v", soft)
	}
	armed := EndTurnGateFor(frame, false, false, true)
	if !armed.Enabled || armed.Soft || armed.Label != "End turn now · Space" {
		t.Fatalf("armed gate = %+v", armed)
	}
	frame.Bands[0].SpatialActionUsed = true
	frame.Bands[1].HasQueuedMigration = true
	clear := EndTurnGateFor(frame, false, false, false)
	if !clear.Enabled || clear.Soft || clear.Label != "End turn · Space" {
		t.Fatalf("clear gate = %+v", clear)
	}
	one := checklistFrame()
	one.Bands[1].SpatialActionUsed = true
	if gate := EndTurnGateFor(one, false, false, false); gate.Label != "End turn · 1 band still needs a move" {
		t.Fatalf("singular gate = %+v", gate)
	}
}

func TestCompassDirectionNamesEightNeighbours(t *testing.T) {
	frame := &gameapi.Frame{Tiles: []gameapi.Tile{{ID: 0, X: 5, Y: 5}, {ID: 1, X: 6, Y: 4}, {ID: 2, X: 5, Y: 6}, {ID: 3, X: 9, Y: 9}}}
	if got := CompassDirection(frame, 0, 1); got != "NE" {
		t.Fatalf("NE = %q", got)
	}
	if got := CompassDirection(frame, 0, 2); got != "S" {
		t.Fatalf("S = %q", got)
	}
	if got := CompassDirection(frame, 0, 3); got != "" {
		t.Fatalf("distant = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui/ -run 'TestDefaultOpenRow|TestSummaries|TestEndTurnGate|TestCompassDirection' -v` — Expected: FAIL, undefined.

- [ ] **Step 3: Implement `pkg/ui/checklist.go`**

```go
package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// ChecklistRow is one of the three per-band decisions the panel walks the
// player through each turn (spec §5.1). Exactly one row is open at a time.
type ChecklistRow uint8

const (
	RowMove ChecklistRow = iota
	RowResearch
	RowWorkforce
	ChecklistRowCount
)

func (row ChecklistRow) Title() string {
	return [...]string{"Move", "Research", "Workforce"}[row]
}

// ResearchDone is the Research row predicate.
func ResearchDone(band gameapi.Band) bool { return band.HasResearchTarget }

// DefaultOpenRow is the first row that still needs a decision, falling back to
// Move. It is recomputed on selection change, load, new campaign, and completed
// turn; the player may then open any row explicitly.
func DefaultOpenRow(band *gameapi.Band) ChecklistRow {
	if band == nil {
		return RowMove
	}
	switch {
	case !MoveDone(*band):
		return RowMove
	case !ResearchDone(*band):
		return RowResearch
	default:
		return RowMove
	}
}

// MoveSummary is the collapsed Move row's one-line state.
func MoveSummary(frame *gameapi.Frame, band gameapi.Band) string {
	switch {
	case band.HasQueuedMigration:
		label := "Move set"
		if frame != nil && int(band.QueuedMigration) < len(frame.Tiles) {
			label += " → " + frame.Tiles[band.QueuedMigration].Biome.String()
			if direction := CompassDirection(frame, band.TileID, band.QueuedMigration); direction != "" {
				label += ", " + direction
			}
		}
		return label
	case band.HasInterbreedTarget:
		return fmt.Sprintf("Interbreeding with B%d", band.InterbreedTargetID)
	case band.SpatialActionUsed:
		// The frame does not say which action consumed it; a split is the only
		// remaining player action that can (DESIGN.md forbids inferring more).
		return "Split queued"
	default:
		return "Choose a destination"
	}
}

// ResearchSummary is the collapsed Research row's one-line state.
func ResearchSummary(band gameapi.Band) string {
	if !band.HasResearchTarget {
		return "No target · choose one"
	}
	option := band.ResearchOptions[band.ResearchTarget]
	return fmt.Sprintf("%s %.0f/%.0f · %+.1f/turn", band.ResearchTarget, band.ResearchProgress[band.ResearchTarget], option.Cost, band.OriginalResearchGainPreview)
}

// RoleShortLabel is the compact per-role label used in summaries and sliders.
func RoleShortLabel(role gameapi.WorkforceRole) string {
	return [...]string{"Foraging", "Hunt / fish", "Toolcraft", "Megafauna", "Shelter / care"}[role]
}

var roleSummaryLabels = [gameapi.AssignmentCount]string{"Forage", "Hunt", "Tools", "Mega", "Shelter"}

// WorkforceSummary is the collapsed Workforce row's one-line state.
func WorkforceSummary(allocation [gameapi.AssignmentCount]uint16, dirty bool) string {
	if dirty {
		return "Unapplied changes"
	}
	summary := ""
	for role, points := range allocation {
		if role > 0 {
			summary += " · "
		}
		summary += fmt.Sprintf("%s %d", roleSummaryLabels[role], points/100)
	}
	return summary
}

// EndTurnGate is the End turn button's state (spec §5.3). Hard blocks disable
// the button; the soft block keeps it enabled but amber and demands a second
// click (or Space) while unarmed.
type EndTurnGate struct {
	Enabled bool
	Soft    bool
	Label   string
}

func EndTurnGateFor(frame *gameapi.Frame, dirtyDraft, pendingCursor, armed bool) EndTurnGate {
	switch {
	case dirtyDraft:
		return EndTurnGate{Label: "End turn · apply or discard workforce changes"}
	case pendingCursor:
		return EndTurnGate{Label: "End turn · queue or clear the arrow-key choice"}
	}
	waiting := 0
	if frame != nil {
		waiting = BandsNeedingMove(frame.Bands)
	}
	switch {
	case waiting == 0:
		return EndTurnGate{Enabled: true, Label: "End turn · Space"}
	case armed:
		return EndTurnGate{Enabled: true, Label: "End turn now · Space"}
	case waiting == 1:
		return EndTurnGate{Enabled: true, Soft: true, Label: "End turn · 1 band still needs a move"}
	default:
		return EndTurnGate{Enabled: true, Soft: true, Label: fmt.Sprintf("End turn · %d bands still need a move", waiting)}
	}
}

// CompassDirection names the one-step direction from one tile to an adjacent
// tile, or returns "" when they are not neighbours. Y grows southward.
func CompassDirection(frame *gameapi.Frame, from, to gameapi.TileID) string {
	if frame == nil || int(from) >= len(frame.Tiles) || int(to) >= len(frame.Tiles) {
		return ""
	}
	dx := frame.Tiles[to].X - frame.Tiles[from].X
	dy := frame.Tiles[to].Y - frame.Tiles[from].Y
	if dx < -1 || dx > 1 || dy < -1 || dy > 1 || (dx == 0 && dy == 0) {
		return ""
	}
	return [3][3]string{{"NW", "N", "NE"}, {"W", "", "E"}, {"SW", "S", "SE"}}[dy+1][dx+1]
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/ui/ -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ui/checklist.go pkg/ui/checklist_test.go
git commit -m "Add checklist row predicates and the End-turn gate

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 4: Tile liveability comparison with tiers

**Files:**
- Create: `pkg/ui/liveability.go`, `pkg/ui/liveability_test.go`
- Leave `pkg/render/tile_info.go` until Task 11.

**Interfaces:**
- Consumes: `render.MigrationPreview{BandID, TileID, Visible}`, `render.TileHover{TileID, Visible}` (existing types).
- Produces:
  - `type TargetSource uint8`: `TargetNone`, `TargetCursor`, `TargetQueued`, `TargetHover`; `func (s TargetSource) Label() string` → "hover or click an outlined tile", "cursor", "queued", "hover".
  - `type LiveabilityTier uint8`: `TierNormal`, `TierAmber`, `TierRed`.
  - `type TileLiveability struct` (exported fields: `Available bool; Status string; Biome, Region string; FoodStock, FoodCap, WaterStock, WaterCap, EcologicalK, BaselineK, Degradation, SeasonalRisk, ChronicRisk, CrowdingDecline, NaturalShelter, MovementCost float64; HasRisk bool; ArchaicBands int; ArchaicPopulation uint64; RequiresPassage bool; Passage gameapi.PassageID; Reachable bool`)
  - `func CurrentTileLiveability(frame *gameapi.Frame, band *gameapi.Band) TileLiveability`
  - `func TargetTile(band *gameapi.Band, preview render.MigrationPreview, hover render.TileHover) (gameapi.TileID, TargetSource)` — precedence cursor > queued > hover.
  - `func TargetTileLiveability(frame *gameapi.Frame, band *gameapi.Band, tile gameapi.TileID) TileLiveability`
  - `type LiveabilityRow struct { Label, Here, Target string; HereTier, TargetTier LiveabilityTier; Delta int }` where `Delta` is +1 better, −1 worse, 0 same or unknown.
  - `func LiveabilityRows(band *gameapi.Band, here, target TileLiveability) []LiveabilityRow` — eight rows: Biome, Food, Capacity, Water, Shelter, Mortality, Route, Archaic.

Tier thresholds are spec §5.4 and must be constants named in the file.

- [ ] **Step 1: Write the failing tests**

Create `pkg/ui/liveability_test.go`:

```go
package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

func liveabilityFrame() *gameapi.Frame {
	tile := func(id gameapi.TileID, food, water float64) gameapi.Tile {
		return gameapi.Tile{
			ID: id, X: int(id), Y: 0, Land: true, Explored: true, Biome: gameapi.Savanna, Region: gameapi.EastAfrica,
			BaselineK: 150, EcologicalK: 150, FloraStock: food, FloraCap: 810, WaterStock: water, WaterCap: 500,
			NaturalShelter: 0.5, MovementCost: 1.2,
		}
	}
	return &gameapi.Frame{
		Tiles: []gameapi.Tile{tile(0, 212, 315), tile(1, 640, 480), {ID: 2, X: 2, Land: true, Explored: false}},
		Bands: []gameapi.Band{
			{
				ID: 3, Species: gameapi.HomoSapiens, Population: 68, TileID: 0, Health: 1,
				SeasonalMortalityRate: 0.0003, ChronicMortalityRate: 0.0016,
				LastFoodReport:        gameapi.FoodTurnReport{Turn: 12, RequiredFU: 300},
				MigrationCandidates:   []gameapi.MigrationCandidate{{TileID: 1, SeasonalMortalityRate: 0.0003, ChronicMortalityRate: 0.002}},
			},
			{ID: 9, Species: gameapi.ArchaicHominin, Population: 40, TileID: 1},
		},
	}
}

func TestTargetTilePrecedenceIsCursorThenQueuedThenHover(t *testing.T) {
	band := &gameapi.Band{ID: 3, HasQueuedMigration: true, QueuedMigration: 5}
	preview := render.MigrationPreview{BandID: 3, TileID: 4, Visible: true}
	hover := render.TileHover{TileID: 6, Visible: true}
	if tile, source := TargetTile(band, preview, hover); tile != 4 || source != TargetCursor {
		t.Fatalf("cursor precedence = (%d, %v)", tile, source)
	}
	if tile, source := TargetTile(band, render.MigrationPreview{BandID: 8, TileID: 4, Visible: true}, hover); tile != 5 || source != TargetQueued {
		t.Fatalf("queued precedence = (%d, %v)", tile, source)
	}
	if tile, source := TargetTile(&gameapi.Band{ID: 3}, render.MigrationPreview{}, hover); tile != 6 || source != TargetHover {
		t.Fatalf("hover = (%d, %v)", tile, source)
	}
	if _, source := TargetTile(&gameapi.Band{ID: 3}, render.MigrationPreview{}, render.TileHover{}); source != TargetNone {
		t.Fatalf("no target = %v", source)
	}
	if TargetNone.Label() != "hover or click an outlined tile" || TargetCursor.Label() != "cursor" || TargetQueued.Label() != "queued" || TargetHover.Label() != "hover" {
		t.Fatal("target source labels drifted from the spec")
	}
}

func TestLiveabilityRowsColorAbsoluteStateAndMarkRelativeDelta(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	here := CurrentTileLiveability(frame, band)
	target := TargetTileLiveability(frame, band, 1)
	if !here.Available || !target.Available || !target.Reachable {
		t.Fatalf("summaries = here %+v target %+v", here, target)
	}
	rows := LiveabilityRows(band, here, target)
	byLabel := map[string]LiveabilityRow{}
	for _, row := range rows {
		byLabel[row.Label] = row
	}
	if len(rows) != 8 {
		t.Fatalf("row count = %d, want 8", len(rows))
	}
	// Food 212 is below last turn's 300 FU requirement: red here, normal there, target better.
	if food := byLabel["Food"]; food.HereTier != TierRed || food.TargetTier != TierNormal || food.Delta != 1 || food.Here != "212 / 810" {
		t.Fatalf("food row = %+v", food)
	}
	// Water 315/500 = 0.63 of cap: normal; 480 is better.
	if water := byLabel["Water"]; water.HereTier != TierNormal || water.Delta != 1 {
		t.Fatalf("water row = %+v", water)
	}
	// Chronic 0.16% -> 0.20%: worse, both below the 0.004 amber tier.
	if mortality := byLabel["Mortality"]; mortality.Delta != -1 || mortality.HereTier != TierNormal {
		t.Fatalf("mortality row = %+v", mortality)
	}
	if archaic := byLabel["Archaic"]; archaic.TargetTier != TierAmber || archaic.Target != "1 band · pop 40" || archaic.Here != "none" {
		t.Fatalf("archaic row = %+v", archaic)
	}
	if route := byLabel["Route"]; route.Target != "×1.20 · 1 turn" || route.Here != "—" {
		t.Fatalf("route row = %+v", route)
	}
}

func TestLiveabilityTiersUseTheSpecThresholds(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	frame.Tiles[0].WaterStock = 120 // 0.24 of cap
	frame.Tiles[0].Degradation = 0.3
	band.SeasonalMortalityRate, band.ChronicMortalityRate = 0.005, 0.004
	frame.Tiles[0].NaturalShelter = 0.2
	rows := LiveabilityRows(band, CurrentTileLiveability(frame, band), TileLiveability{})
	byLabel := map[string]LiveabilityRow{}
	for _, row := range rows {
		byLabel[row.Label] = row
	}
	if byLabel["Water"].HereTier != TierRed || byLabel["Capacity"].HereTier != TierAmber || byLabel["Mortality"].HereTier != TierRed || byLabel["Shelter"].HereTier != TierAmber {
		t.Fatalf("tiers = water %v capacity %v mortality %v shelter %v", byLabel["Water"].HereTier, byLabel["Capacity"].HereTier, byLabel["Mortality"].HereTier, byLabel["Shelter"].HereTier)
	}
	if byLabel["Biome"].Target != "—" {
		t.Fatalf("unavailable target biome = %q, want an em dash", byLabel["Biome"].Target)
	}
}

func TestTargetTileLiveabilityHidesFogAndExplainsUnreachable(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	if fog := TargetTileLiveability(frame, band, 2); fog.Available || fog.Status != "Unexplored · details hidden" {
		t.Fatalf("fog summary = %+v", fog)
	}
	frame.Tiles[1].Explored = true
	band.MigrationCandidates = nil
	if far := TargetTileLiveability(frame, band, 1); !far.Available || far.Reachable || far.Status != "not reachable" {
		t.Fatalf("unreachable summary = %+v", far)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui/ -run 'TestTargetTile|TestLiveability' -v` — Expected: FAIL, undefined.

- [ ] **Step 3: Implement `pkg/ui/liveability.go`**

```go
package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

// Presentation-only liveability tiers (spec §5.4). Never simulation inputs.
const (
	foodAmberRequirementFactor = 1.5  // amber below 1.5 × last turn's RequiredFU
	waterAmberCapFraction      = 0.5  // amber below half of cap
	waterRedCapFraction        = 0.25 // red below a quarter of cap
	degradationAmber           = 0.25
	degradationRed             = 0.5
	mortalityAmber             = bandDangerMortalityRate // 0.004, shared with the danger tier
	mortalityRed               = 0.008
	shelterAmber               = 0.3
)

type TargetSource uint8

const (
	TargetNone TargetSource = iota
	TargetCursor
	TargetQueued
	TargetHover
)

func (source TargetSource) Label() string {
	return [...]string{"hover or click an outlined tile", "cursor", "queued", "hover"}[source]
}

type LiveabilityTier uint8

const (
	TierNormal LiveabilityTier = iota
	TierAmber
	TierRed
)

// TileLiveability is a presentation-safe reading of one explored land tile.
type TileLiveability struct {
	Available         bool
	Status            string
	Biome             string
	Region            string
	FoodStock         float64
	FoodCap           float64
	WaterStock        float64
	WaterCap          float64
	EcologicalK       float64
	BaselineK         float64
	Degradation       float64
	SeasonalRisk      float64
	ChronicRisk       float64
	CrowdingDecline   float64
	HasRisk           bool
	NaturalShelter    float64
	MovementCost      float64
	ArchaicBands      int
	ArchaicPopulation uint64
	RequiresPassage   bool
	Passage           gameapi.PassageID
	Reachable         bool
}

// CurrentTileLiveability reads the band's own tile with its projected rates.
func CurrentTileLiveability(frame *gameapi.Frame, band *gameapi.Band) TileLiveability {
	if band == nil {
		return TileLiveability{Status: "No active band"}
	}
	summary := summarizeTile(frame, band.TileID)
	if summary.Available {
		summary.Status = "current"
		summary.SeasonalRisk, summary.ChronicRisk, summary.HasRisk = band.SeasonalMortalityRate, band.ChronicMortalityRate, true
	}
	return summary
}

// TargetTile applies the spec's precedence: an arrow-key cursor for this band,
// then an already queued migration, then pointer hover.
func TargetTile(band *gameapi.Band, preview render.MigrationPreview, hover render.TileHover) (gameapi.TileID, TargetSource) {
	if band == nil {
		return 0, TargetNone
	}
	switch {
	case preview.Visible && preview.BandID == band.ID:
		return preview.TileID, TargetCursor
	case band.HasQueuedMigration:
		return band.QueuedMigration, TargetQueued
	case hover.Visible:
		return hover.TileID, TargetHover
	default:
		return 0, TargetNone
	}
}

// TargetTileLiveability reads a candidate tile, taking route-dependent rates
// from the authoritative MigrationCandidates entry when the tile is reachable.
func TargetTileLiveability(frame *gameapi.Frame, band *gameapi.Band, tile gameapi.TileID) TileLiveability {
	summary := summarizeTile(frame, tile)
	if !summary.Available || band == nil {
		return summary
	}
	for _, candidate := range band.MigrationCandidates {
		if candidate.TileID != tile {
			continue
		}
		summary.Reachable = true
		summary.Status = "reachable"
		summary.RequiresPassage, summary.Passage = candidate.RequiresPassage, candidate.Passage
		if candidate.RequiresPassage {
			summary.Status += " via " + candidate.Passage.String()
		}
		summary.SeasonalRisk, summary.ChronicRisk, summary.HasRisk = candidate.SeasonalMortalityRate, candidate.ChronicMortalityRate, true
		summary.CrowdingDecline = candidate.CrowdingDecline
		return summary
	}
	if gameapi.EscarpmentBlocks(frame, band.TileID, tile) {
		summary.Status = "escarpment blocks approach"
	} else {
		summary.Status = "not reachable"
	}
	return summary
}

func summarizeTile(frame *gameapi.Frame, tileID gameapi.TileID) TileLiveability {
	summary := TileLiveability{Status: "Invalid tile"}
	if frame == nil || int(tileID) >= len(frame.Tiles) {
		return summary
	}
	tile := frame.Tiles[tileID]
	switch {
	case !tile.Explored:
		summary.Status = "Unexplored · details hidden"
		return summary
	case !tile.Land:
		summary.Status = "Open water · cannot occupy"
		return summary
	case tile.BaselineK <= 0:
		summary.Status = "Uninhabitable in this climate"
		return summary
	}
	summary.Available = true
	summary.Biome, summary.Region = tile.Biome.String(), tile.Region.String()
	summary.FoodStock, summary.FoodCap = tile.FloraStock+tile.FaunaStock, tile.FloraCap+tile.FaunaCap
	summary.WaterStock, summary.WaterCap = tile.WaterStock, tile.WaterCap
	summary.EcologicalK, summary.BaselineK, summary.Degradation = tile.EcologicalK, tile.BaselineK, tile.Degradation
	summary.NaturalShelter, summary.MovementCost = tile.NaturalShelter, tile.MovementCost
	for _, band := range frame.Bands {
		if band.Species == gameapi.ArchaicHominin && band.TileID == tileID && band.Population > 0 {
			summary.ArchaicBands++
			summary.ArchaicPopulation += uint64(band.Population)
		}
	}
	return summary
}

// LiveabilityRow is one HERE / TARGET comparison line for the Move row.
type LiveabilityRow struct {
	Label      string
	Here       string
	Target     string
	HereTier   LiveabilityTier
	TargetTier LiveabilityTier
	Delta      int // +1 target better, -1 worse, 0 same or unknown
}

// LiveabilityRows builds the eight comparison rows. Absolute tiers color each
// side; Delta compares target to here where both are available.
func LiveabilityRows(band *gameapi.Band, here, target TileLiveability) []LiveabilityRow {
	required := 0.0
	if band != nil && band.LastFoodReport.Turn > 0 {
		required = band.LastFoodReport.RequiredFU
	}
	both := here.Available && target.Available
	row := func(label string, value func(TileLiveability) string, tier func(TileLiveability) LiveabilityTier, higherIsBetter bool, metric func(TileLiveability) float64) LiveabilityRow {
		result := LiveabilityRow{Label: label, Here: "—", Target: "—"}
		if here.Available {
			result.Here, result.HereTier = value(here), tier(here)
		}
		if target.Available {
			result.Target, result.TargetTier = value(target), tier(target)
		}
		if both && metric != nil {
			switch h, t := metric(here), metric(target); {
			case t > h && higherIsBetter, t < h && !higherIsBetter:
				result.Delta = 1
			case t != h:
				result.Delta = -1
			}
		}
		return result
	}
	normal := func(TileLiveability) LiveabilityTier { return TierNormal }
	foodTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case required > 0 && s.FoodStock < required:
			return TierRed
		case required > 0 && s.FoodStock < foodAmberRequirementFactor*required:
			return TierAmber
		}
		return TierNormal
	}
	waterTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case s.WaterCap > 0 && s.WaterStock < waterRedCapFraction*s.WaterCap:
			return TierRed
		case s.WaterCap > 0 && s.WaterStock < waterAmberCapFraction*s.WaterCap:
			return TierAmber
		}
		return TierNormal
	}
	capacityTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case s.Degradation >= degradationRed:
			return TierRed
		case s.Degradation >= degradationAmber:
			return TierAmber
		}
		return TierNormal
	}
	mortalityTier := func(s TileLiveability) LiveabilityTier {
		total := s.SeasonalRisk + s.ChronicRisk
		switch {
		case !s.HasRisk:
			return TierNormal
		case total >= mortalityRed:
			return TierRed
		case total >= mortalityAmber:
			return TierAmber
		}
		return TierNormal
	}
	shelterTier := func(s TileLiveability) LiveabilityTier {
		if s.NaturalShelter < shelterAmber {
			return TierAmber
		}
		return TierNormal
	}
	archaicTier := func(s TileLiveability) LiveabilityTier {
		if s.ArchaicBands > 0 {
			return TierAmber
		}
		return TierNormal
	}
	mortalityValue := func(s TileLiveability) string {
		switch {
		case s.CrowdingDecline > 0:
			return fmt.Sprintf("crowding −%.0f · %.2f%%", s.CrowdingDecline, (s.SeasonalRisk+s.ChronicRisk)*100)
		case s.HasRisk:
			return fmt.Sprintf("%.2f%% · chr %.2f%%", s.SeasonalRisk*100, s.ChronicRisk*100)
		default:
			return "unavailable"
		}
	}
	routeValue := func(s TileLiveability) string {
		if !s.Reachable {
			return "—"
		}
		if s.RequiresPassage {
			return "via " + s.Passage.String()
		}
		return fmt.Sprintf("×%.2f · 1 turn", s.MovementCost)
	}
	archaicValue := func(s TileLiveability) string {
		if s.ArchaicBands == 0 {
			return "none"
		}
		noun := "band"
		if s.ArchaicBands != 1 {
			noun = "bands"
		}
		return fmt.Sprintf("%d %s · pop %d", s.ArchaicBands, noun, s.ArchaicPopulation)
	}
	return []LiveabilityRow{
		row("Biome", func(s TileLiveability) string { return s.Biome }, normal, true, nil),
		row("Food", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.FoodStock, s.FoodCap) }, foodTier, true, func(s TileLiveability) float64 { return s.FoodStock }),
		row("Capacity", func(s TileLiveability) string { return fmt.Sprintf("%.0f · %.0f%% degr.", s.EcologicalK, s.Degradation*100) }, capacityTier, true, func(s TileLiveability) float64 { return s.EcologicalK }),
		row("Water", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.WaterStock, s.WaterCap) }, waterTier, true, func(s TileLiveability) float64 { return s.WaterStock }),
		row("Shelter", func(s TileLiveability) string { return fmt.Sprintf("%.0f%%", s.NaturalShelter*100) }, shelterTier, true, func(s TileLiveability) float64 { return s.NaturalShelter }),
		row("Mortality", mortalityValue, mortalityTier, false, func(s TileLiveability) float64 { return s.SeasonalRisk + s.ChronicRisk + s.CrowdingDecline }),
		row("Route", routeValue, normal, true, nil),
		row("Archaic", archaicValue, archaicTier, true, nil),
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/ui/ -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ui/liveability.go pkg/ui/liveability_test.go
git commit -m "Add HERE/TARGET liveability rows with presentation tiers

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 5: First-turn guide state machine

**Files:**
- Create: `pkg/ui/guide.go`, `pkg/ui/guide_test.go`

**Interfaces:**
- Produces:
  - `type GuideStep uint8`: `GuideMove`, `GuideResearch`, `GuideWorkforce`, `GuideEndTurn`, `GuideClosing`, `GuideDismissed`.
  - `type GuideState struct { Step GuideStep }` (comparable).
  - `func NewGuideState(dismissed bool) GuideState`
  - `func (g GuideState) Visible() bool`
  - `func (g GuideState) Next() GuideState` — advances one step; Closing → Closing (only Dismiss leaves).
  - `func (g GuideState) Dismiss() GuideState`
  - `func (g GuideState) Observe(band *gameapi.Band) GuideState` — advances Move/Research steps when their Done predicate is true for the given band.
  - `func (g GuideState) ObserveTurnCompleted() GuideState` — EndTurn → Closing.
  - `func (g GuideState) Progress() (current, total int)` — 1..4 of 4; Closing reports (4, 4).
  - `func (g GuideState) Title() string`, `func (g GuideState) Body() string` — the card copy.

- [ ] **Step 1: Write the failing tests**

Create `pkg/ui/guide_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestGuideAdvancesOnDonePredicatesAndNextButOnlyDismissEnds(t *testing.T) {
	guide := NewGuideState(false)
	if !guide.Visible() || guide.Step != GuideMove {
		t.Fatalf("fresh guide = %+v", guide)
	}
	band := &gameapi.Band{Species: gameapi.HomoSapiens}
	if guide = guide.Observe(band); guide.Step != GuideMove {
		t.Fatal("guide advanced without a move")
	}
	band.HasQueuedMigration = true
	if guide = guide.Observe(band); guide.Step != GuideResearch {
		t.Fatalf("after move step = %v, want Research", guide.Step)
	}
	// Research not chosen yet: stays; Next skips it explicitly.
	if guide = guide.Observe(band); guide.Step != GuideResearch {
		t.Fatal("research step advanced without a target")
	}
	guide = guide.Next()
	if guide.Step != GuideWorkforce {
		t.Fatalf("after Next = %v, want Workforce", guide.Step)
	}
	guide = guide.Next()
	if guide.Step != GuideEndTurn {
		t.Fatalf("after second Next = %v, want EndTurn", guide.Step)
	}
	if guide = guide.ObserveTurnCompleted(); guide.Step != GuideClosing {
		t.Fatalf("after turn = %v, want Closing", guide.Step)
	}
	if guide = guide.Next(); guide.Step != GuideClosing || !guide.Visible() {
		t.Fatal("Next left the closing card; only the × may")
	}
	if guide = guide.ObserveTurnCompleted(); guide.Step != GuideClosing {
		t.Fatal("a later turn changed the closing card")
	}
	if guide = guide.Dismiss(); guide.Visible() || guide.Step != GuideDismissed {
		t.Fatalf("dismissed = %+v", guide)
	}
	if guide = guide.Observe(band).Next().ObserveTurnCompleted(); guide.Step != GuideDismissed {
		t.Fatal("dismissed guide came back")
	}
	if NewGuideState(true).Visible() {
		t.Fatal("persisted dismissal was ignored")
	}
}

func TestGuideCopyAndProgress(t *testing.T) {
	steps := []GuideStep{GuideMove, GuideResearch, GuideWorkforce, GuideEndTurn}
	for index, step := range steps {
		guide := GuideState{Step: step}
		current, total := guide.Progress()
		if current != index+1 || total != 4 {
			t.Fatalf("progress for %v = %d/%d", step, current, total)
		}
		if guide.Title() == "" || guide.Body() == "" {
			t.Fatalf("step %v has empty copy", step)
		}
	}
	closing := GuideState{Step: GuideClosing}
	if current, total := closing.Progress(); current != 4 || total != 4 {
		t.Fatalf("closing progress = %d/%d", current, total)
	}
	if !strings.Contains(closing.Body(), "Tab") || !strings.Contains(closing.Body(), "×") {
		t.Fatalf("closing copy = %q, want Tab and × mentioned", closing.Body())
	}
	if !strings.Contains(GuideState{Step: GuideMove}.Body(), "gold") {
		t.Fatal("move step copy does not mention the gold outline")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui/ -run TestGuide -v` — Expected: FAIL, undefined.

- [ ] **Step 3: Implement `pkg/ui/guide.go`**

```go
package ui

import "github.com/adsouza/africa2ice/pkg/gameapi"

// GuideStep is the first-turn guide's position (spec §7). The guide never
// gates input: it advances when the player does the step or clicks Next, and
// only the × dismisses it.
type GuideStep uint8

const (
	GuideMove GuideStep = iota
	GuideResearch
	GuideWorkforce
	GuideEndTurn
	GuideClosing
	GuideDismissed
)

// GuideState is UI-local; only its dismissal is persisted as a preference.
type GuideState struct {
	Step GuideStep
}

func NewGuideState(dismissed bool) GuideState {
	if dismissed {
		return GuideState{Step: GuideDismissed}
	}
	return GuideState{Step: GuideMove}
}

func (guide GuideState) Visible() bool { return guide.Step != GuideDismissed }

// Next moves forward one step; the closing card stays until dismissed.
func (guide GuideState) Next() GuideState {
	if guide.Step < GuideClosing {
		guide.Step++
	}
	return guide
}

func (guide GuideState) Dismiss() GuideState { return GuideState{Step: GuideDismissed} }

// Observe advances the Move and Research steps when the selected band's row
// predicate flips true. Workforce is optional and advances only on Next.
func (guide GuideState) Observe(band *gameapi.Band) GuideState {
	if band == nil {
		return guide
	}
	switch {
	case guide.Step == GuideMove && MoveDone(*band):
		return guide.Next()
	case guide.Step == GuideResearch && ResearchDone(*band):
		return guide.Next()
	}
	return guide
}

// ObserveTurnCompleted moves the End turn step to the closing card.
func (guide GuideState) ObserveTurnCompleted() GuideState {
	if guide.Step == GuideEndTurn {
		return guide.Next()
	}
	return guide
}

// Progress is the 1-based step of four shown on the card; Closing reports 4/4.
func (guide GuideState) Progress() (int, int) {
	if guide.Step >= GuideClosing {
		return 4, 4
	}
	return int(guide.Step) + 1, 4
}

func (guide GuideState) Title() string {
	switch guide.Step {
	case GuideMove:
		return "FIRST TURN · STEP 1 OF 4 · MOVE"
	case GuideResearch:
		return "FIRST TURN · STEP 2 OF 4 · RESEARCH"
	case GuideWorkforce:
		return "FIRST TURN · STEP 3 OF 4 · WORKFORCE"
	case GuideEndTurn:
		return "FIRST TURN · STEP 4 OF 4 · END TURN"
	case GuideClosing:
		return "FIRST TURN · THAT'S A FULL TURN"
	default:
		return ""
	}
}

func (guide GuideState) Body() string {
	switch guide.Step {
	case GuideMove:
		return "Each turn, every band may make one move. This band's reachable tiles are outlined on the map; the gold one has the best food. Click it, or click Move to gold tile below. Staying put is also fine."
	case GuideResearch:
		return "Pick a technology for this band to work toward. Firecraft, Hafted Tools and Plant Knowledge need nothing first. Progress accrues every turn."
	case GuideWorkforce:
		return "The five sliders share out the band's labor. The defaults are sound for now; come back when food runs short. Changes apply only when the total is 100%."
	case GuideEndTurn:
		return "Repeat for each band, then end the turn with the button or Space. Nothing happens until you do."
	case GuideClosing:
		return "Tab moves between bands. A gold chip still needs a move; green is done; ! and !! mark bands in trouble. Close this card with the × whenever you like."
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/ui/ -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ui/guide.go pkg/ui/guide_test.go
git commit -m "Add the first-turn guide state machine

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 6: UI settings schema 2

**Files:**
- Modify: `pkg/ui/ui_settings.go`, `pkg/ui/ui_settings_test.go`

**Interfaces:**
- Produces: `UISettings` gains `GuideDismissed bool` and `FieldNotesExpanded bool` (JSON names identical); `UISettingsSchemaVersion = 2`; `DecodeUISettings` accepts schema 1 (new fields false) and 2 (all six fields required).

- [ ] **Step 1: Write the failing tests**

Append to `pkg/ui/ui_settings_test.go`:

```go
func TestUISettingsSchemaTwoRoundTripsAndUpgradesSchemaOne(t *testing.T) {
	want := UISettings{SchemaVersion: 2, FieldNotesVisible: false, MasterVolume: 0.3, Muted: true, GuideDismissed: true, FieldNotesExpanded: true}
	payload, err := EncodeUISettings(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeUISettings(payload)
	if err != nil || got != want {
		t.Fatalf("round trip = %+v, %v; want %+v", got, err, want)
	}

	v1 := []byte(`{"SchemaVersion":1,"FieldNotesVisible":false,"MasterVolume":0.3,"Muted":true}`)
	upgraded, err := DecodeUISettings(v1)
	if err != nil {
		t.Fatalf("schema 1 payload rejected: %v", err)
	}
	if upgraded.SchemaVersion != 2 || upgraded.GuideDismissed || upgraded.FieldNotesExpanded || upgraded.MasterVolume != 0.3 || upgraded.FieldNotesVisible || !upgraded.Muted {
		t.Fatalf("upgraded schema 1 = %+v", upgraded)
	}

	missing := []byte(`{"SchemaVersion":2,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false}`)
	if _, err := DecodeUISettings(missing); err == nil {
		t.Fatal("schema 2 payload without FieldNotesExpanded was accepted")
	}
	if _, err := DecodeUISettings([]byte(`{"SchemaVersion":3,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false}`)); err == nil {
		t.Fatal("unknown schema 3 was accepted")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/ui/ -run TestUISettings -v` — Expected: FAIL (unknown fields / schema 2 unsupported).

- [ ] **Step 3: Implement**

In `pkg/ui/ui_settings.go`:

```go
const UISettingsSchemaVersion = 2

type UISettings struct {
	SchemaVersion      int     `json:"SchemaVersion"`
	FieldNotesVisible  bool    `json:"FieldNotesVisible"`
	MasterVolume       float64 `json:"MasterVolume"`
	Muted              bool    `json:"Muted"`
	GuideDismissed     bool    `json:"GuideDismissed"`
	FieldNotesExpanded bool    `json:"FieldNotesExpanded"`
}
```

Replace the body of `DecodeUISettings` after the trailing-value check with:

```go
	var settings UISettings
	if raw, ok := fields["SchemaVersion"]; !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return DefaultUISettings(), errors.New("decode UI settings: missing or null SchemaVersion")
	} else if err := json.Unmarshal(raw, &settings.SchemaVersion); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode UI settings schema: %w", err)
	}
	required := []string{"FieldNotesVisible", "MasterVolume", "Muted"}
	switch settings.SchemaVersion {
	case 1:
		// Schema 1 predates the guide and drawer height; both default to false.
	case 2:
		required = append(required, "GuideDismissed", "FieldNotesExpanded")
	default:
		return DefaultUISettings(), fmt.Errorf("unsupported UI settings schema %d", settings.SchemaVersion)
	}
	for _, name := range required {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return DefaultUISettings(), fmt.Errorf("decode UI settings: missing or null %s", name)
		}
	}
	if err := json.Unmarshal(fields["FieldNotesVisible"], &settings.FieldNotesVisible); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode FieldNotesVisible: %w", err)
	}
	if err := json.Unmarshal(fields["MasterVolume"], &settings.MasterVolume); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode MasterVolume: %w", err)
	}
	if math.IsNaN(settings.MasterVolume) || math.IsInf(settings.MasterVolume, 0) {
		return DefaultUISettings(), errors.New("decode MasterVolume: value must be finite")
	}
	if err := json.Unmarshal(fields["Muted"], &settings.Muted); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode Muted: %w", err)
	}
	if settings.SchemaVersion == 2 {
		if err := json.Unmarshal(fields["GuideDismissed"], &settings.GuideDismissed); err != nil {
			return DefaultUISettings(), fmt.Errorf("decode GuideDismissed: %w", err)
		}
		if err := json.Unmarshal(fields["FieldNotesExpanded"], &settings.FieldNotesExpanded); err != nil {
			return DefaultUISettings(), fmt.Errorf("decode FieldNotesExpanded: %w", err)
		}
	}
	return NormalizeUISettings(settings), nil
```

`NormalizeUISettings` already stamps `SchemaVersion = UISettingsSchemaVersion`, which upgrades a decoded v1 record to 2.

- [ ] **Step 4: Run tests and fix the existing v1 test**

Run: `go test ./pkg/ui/ -run TestUISettings -v`. `TestUISettingsRequireCompleteTypedV1Record` may assert an encoded `SchemaVersion` of 1 or reject a payload that is now a valid v1 record; update those expectations to schema 2 output and keep its missing-field and wrong-type cases. Expected after the fix: PASS. Also run `go test ./pkg/app/` (the settings stub round-trips values) — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ui/ui_settings.go pkg/ui/ui_settings_test.go
git commit -m "UI settings schema 2: guide dismissal and drawer height

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

Phase 1 ends here. `go build ./... && go vet ./... && go test ./... && golangci-lint run` must be green.

---

## Phase 2 — Panel on ebitenui, HUD removal, input routing

### Task 7: `pkg/hud` skeleton — State, Intent, theme, fixed layout, Panel lifecycle

**Files:**
- Create: `pkg/hud/state.go`, `pkg/hud/intent.go`, `pkg/hud/theme.go`, `pkg/hud/fixed_layout.go`, `pkg/hud/panel.go`, `pkg/hud/fixed_layout_test.go`, `pkg/hud/panel_test.go`
- Modify: `pkg/hud/doc.go` (remove the blank import)

**Interfaces:**
- Consumes: `ui.ChecklistRow`, `ui.GuideState`, `ui.EndTurnGate`, `ui.SceneID`, `render.MigrationPreview`, `render.TileHover`, `render.FieldNote`, `render.EndScene`, `render.Viewport`, `render.PresentationTransform`.
- Produces (used by every later task):

```go
// state.go
type NotesMode uint8            // NotesHidden, NotesCompact, NotesExpanded
type WorkforceDraft struct {
	Visible      bool
	Population   uint32
	AllocationBP [gameapi.AssignmentCount]uint16
	SelectedRole gameapi.WorkforceRole
	Dirty, Valid bool
}
type CameraState struct{ FocusAvailable, Focused bool }
type StorageRow struct{ Label, Detail string; Slot int; Occupied, Writable bool }
type OverlayState struct {          // scene overlays (Task 13)
	Scene            ui.SceneID
	StorageHeading   string
	StorageRows      [7]StorageRow
	StorageBusy      string           // "" or the pending-operation text
	SettingsDisabled bool
	MasterVolume     float64
	Muted            bool
}
type State struct {
	Frame           *gameapi.Frame
	SelectedBand    gameapi.BandID
	Preview         render.MigrationPreview
	Hover           render.TileHover
	OpenRow         ui.ChecklistRow
	DetailsOpen     bool
	Workforce       WorkforceDraft
	InterbreedFocus gameapi.BandID
	EndTurn         ui.EndTurnGate
	Note            render.FieldNote
	NotesMode       NotesMode
	Guide           ui.GuideState
	Camera          CameraState
	Overlay         OverlayState
	Ending          render.EndScene
	Viewport        render.Viewport
	Transform       render.PresentationTransform
}
// intent.go
type IntentKind uint8
const (
	IntentNone IntentKind = iota
	IntentSelectBand; IntentOpenRow; IntentToggleDetails; IntentSetNotesMode
	IntentMoveTo; IntentMoveToBest; IntentSplit; IntentInterbreed
	IntentChooseResearch
	IntentSelectRole; IntentAdjustRole; IntentApplyWorkforce; IntentDiscardWorkforce
	IntentEndTurn
	IntentGuideNext; IntentGuideDismiss
	IntentCameraToggle
	IntentFocusTrait; IntentFocusEvent; IntentScrollNotes
	IntentOpenMenu; IntentBack; IntentContinue; IntentNewCampaign; IntentOpenStorage; IntentOpenSettings
	IntentReturnToTitle; IntentSaveSlot; IntentLoadSlot; IntentDeleteSlot; IntentSetVolume; IntentToggleMute; IntentShowGuide
	IntentKindCount
)
type Intent struct {
	Kind   IntentKind
	Band   gameapi.BandID
	Tile   gameapi.TileID
	Row    ui.ChecklistRow
	Tech   gameapi.Tech
	Role   gameapi.WorkforceRole
	Trait  gameapi.HeritableTrait
	Event  gameapi.EventKind
	Notes  NotesMode
	Delta  int
	Slot   int
	Volume float64
	Force  bool
	Save   bool // IntentOpenStorage: true = save browser
}
// panel.go
func New() *Panel
func (p *Panel) Update(state State) []Intent
func (p *Panel) Draw(screen *ebiten.Image)
func (p *Panel) Hovered() bool
```

- [ ] **Step 1: Write the failing fixed-layout test**

Create `pkg/hud/fixed_layout_test.go`:

```go
package hud

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestFixedLayoutPlacesChildrenAtTheirOwnRectangles(t *testing.T) {
	root := widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}))
	first := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(fixedRect(image.Rect(10, 20, 110, 220)))))
	second := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(fixedRect(image.Rect(500, 0, 900, 50)))))
	root.AddChild(first, second)
	root.SetLocation(image.Rect(0, 0, 1280, 720))
	root.RequestRelayout()
	root.Update(&widget.UpdateObject{}) // performs the pending relayout
	if got := first.GetWidget().Rect; got != image.Rect(10, 20, 110, 220) {
		t.Fatalf("first child rect = %v", got)
	}
	if got := second.GetWidget().Rect; got != image.Rect(500, 0, 900, 50) {
		t.Fatalf("second child rect = %v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run TestFixedLayout -v` — Expected: FAIL, undefined `fixedLayout`.

- [ ] **Step 3: Implement `pkg/hud/fixed_layout.go`**

```go
package hud

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
)

// fixedRect is the layout data a direct child of a fixedLayout container
// carries: its absolute rectangle in render pixels. The chrome has a handful
// of regions at spec-fixed DIP positions; a layout that just honours them is
// simpler than expressing the panel as nested anchors.
type fixedRect image.Rectangle

type fixedLayout struct{}

func (fixedLayout) PreferredSize(widgets []widget.PreferredSizeLocateableWidget) (int, int) {
	width, height := 0, 0
	for _, child := range widgets {
		if rect, ok := child.GetWidget().LayoutData.(fixedRect); ok {
			width = max(width, rect.Max.X)
			height = max(height, rect.Max.Y)
		}
	}
	return width, height
}

func (fixedLayout) Layout(widgets []widget.PreferredSizeLocateableWidget, _ image.Rectangle) {
	for _, child := range widgets {
		if rect, ok := child.GetWidget().LayoutData.(fixedRect); ok {
			child.SetLocation(image.Rectangle(rect))
		}
	}
}

var _ widget.Layouter = fixedLayout{}
```

If `Update(&widget.UpdateObject{})` does not trigger the relayout in this ebitenui version, call `root.Render(ebiten.NewImage(1280, 720))` in the test instead; rendering performs any pending layout.

Run: `go test ./pkg/hud/ -run TestFixedLayout -v` — Expected: PASS.

- [ ] **Step 4: Write `state.go` and `intent.go`**

`pkg/hud/state.go`:

```go
package hud

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// NotesMode is the Field Notes drawer's height state (spec §4).
type NotesMode uint8

const (
	NotesHidden NotesMode = iota
	NotesCompact
	NotesExpanded
)

// WorkforceDraft mirrors the application's UI-local allocation draft.
type WorkforceDraft struct {
	Visible      bool
	Population   uint32
	AllocationBP [gameapi.AssignmentCount]uint16
	SelectedRole gameapi.WorkforceRole
	Dirty        bool
	Valid        bool
}

// CameraState tells the panel whether the Focus toggle applies (spec §6).
type CameraState struct {
	FocusAvailable bool
	Focused        bool
}

// StorageRow is one save-slot line in the storage browser overlay.
type StorageRow struct {
	Label    string
	Detail   string
	Slot     int
	Occupied bool
	Writable bool
}

// OverlayState describes the modal scene the panel must draw, if any.
type OverlayState struct {
	Scene            ui.SceneID
	StorageHeading   string
	StorageRows      [7]StorageRow
	StorageBusy      string
	SettingsDisabled bool
	MasterVolume     float64
	Muted            bool
}

// State is everything the chrome draws. The application derives it every
// tick; the panel never stores anything the frame or UI-local fields do not
// already hold. It is comparable so the panel can rebuild only on change.
type State struct {
	Frame           *gameapi.Frame
	SelectedBand    gameapi.BandID
	Preview         render.MigrationPreview
	Hover           render.TileHover
	OpenRow         ui.ChecklistRow
	DetailsOpen     bool
	Workforce       WorkforceDraft
	InterbreedFocus gameapi.BandID
	EndTurn         ui.EndTurnGate
	Note            render.FieldNote
	NotesMode       NotesMode
	Guide           ui.GuideState
	Camera          CameraState
	Overlay         OverlayState
	Ending          render.EndScene
	Viewport        render.Viewport
	Transform       render.PresentationTransform
}

// selectedBand returns the selected band within the frame, or nil.
func (state State) selectedBand() *gameapi.Band {
	if state.Frame == nil {
		return nil
	}
	for index := range state.Frame.Bands {
		if state.Frame.Bands[index].ID == state.SelectedBand {
			return &state.Frame.Bands[index]
		}
	}
	return nil
}
```

`pkg/hud/intent.go`:

```go
package hud

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// IntentKind names a player request the chrome produced. The application maps
// each onto the same guarded method its hotkey calls, so mouse and keyboard
// can never diverge.
type IntentKind uint8

const (
	IntentNone IntentKind = iota
	IntentSelectBand
	IntentOpenRow
	IntentToggleDetails
	IntentSetNotesMode
	IntentMoveTo
	IntentMoveToBest
	IntentSplit
	IntentInterbreed
	IntentChooseResearch
	IntentSelectRole
	IntentAdjustRole
	IntentApplyWorkforce
	IntentDiscardWorkforce
	IntentEndTurn
	IntentGuideNext
	IntentGuideDismiss
	IntentCameraToggle
	IntentFocusTrait
	IntentFocusEvent
	IntentScrollNotes
	IntentOpenMenu
	IntentBack
	IntentContinue
	IntentNewCampaign
	IntentOpenStorage
	IntentOpenSettings
	IntentReturnToTitle
	IntentSaveSlot
	IntentLoadSlot
	IntentDeleteSlot
	IntentSetVolume
	IntentToggleMute
	IntentShowGuide
	IntentKindCount
)

// Intent is a kind plus whichever payload fields that kind uses.
type Intent struct {
	Kind   IntentKind
	Band   gameapi.BandID
	Tile   gameapi.TileID
	Row    ui.ChecklistRow
	Tech   gameapi.Tech
	Role   gameapi.WorkforceRole
	Trait  gameapi.HeritableTrait
	Event  gameapi.EventKind
	Notes  NotesMode
	Delta  int
	Slot   int
	Volume float64
	Force  bool
	Save   bool
}
```

- [ ] **Step 5: Write `theme.go`**

```go
package hud

import (
	"bytes"
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// Chrome palette. Map colors stay in pkg/render; these are panel-only.
var (
	colorPanel       = color.RGBA{R: 25, G: 35, B: 42, A: 238}
	colorPanelEdge   = color.RGBA{R: 58, G: 76, B: 82, A: 210}
	colorRow         = color.RGBA{R: 20, G: 29, B: 35, A: 255}
	colorRowOpen     = color.RGBA{R: 28, G: 40, B: 48, A: 255}
	colorTitle       = color.RGBA{R: 239, G: 220, B: 178, A: 255}
	colorText        = color.RGBA{R: 223, G: 229, B: 225, A: 255}
	colorDim         = color.RGBA{R: 159, G: 177, B: 174, A: 255}
	colorGold        = color.RGBA{R: 245, G: 202, B: 92, A: 255}
	colorGoldDeep    = color.RGBA{R: 203, G: 172, B: 104, A: 255}
	colorGreen       = color.RGBA{R: 121, G: 195, B: 137, A: 255}
	colorCyan        = color.RGBA{R: 87, G: 211, B: 211, A: 255}
	colorAmber       = color.RGBA{R: 237, G: 176, B: 84, A: 255}
	colorRed         = color.RGBA{R: 247, G: 137, B: 119, A: 255}
	colorInterbreed  = color.RGBA{R: 186, G: 148, B: 232, A: 255}
	colorQueued      = color.RGBA{R: 255, G: 106, B: 91, A: 255}
	colorButtonIdle  = color.RGBA{R: 35, G: 51, B: 58, A: 255}
	colorButtonHover = color.RGBA{R: 48, G: 68, B: 78, A: 255}
	colorButtonDown  = color.RGBA{R: 24, G: 36, B: 42, A: 255}
	colorDisabled    = color.RGBA{R: 92, G: 106, B: 109, A: 255}
	colorGuide       = color.RGBA{R: 42, G: 36, B: 22, A: 255}
	colorDrawer      = color.RGBA{R: 18, G: 28, B: 34, A: 240}
	colorCelebrate   = color.RGBA{R: 45, G: 39, B: 24, A: 255}
	colorBlack       = color.RGBA{R: 17, G: 17, B: 17, A: 255}
)

// theme owns the font source and the current presentation scale. Every size
// and inset the panel uses goes through px() or face(), so a viewport change
// only needs a rebuild with a new scale.
type theme struct {
	source *text.GoTextFaceSource
	scale  float64
	faces  map[float64]*text.Face
}

func newTheme() *theme {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	return &theme{source: source, scale: 1, faces: map[float64]*text.Face{}}
}

func (t *theme) setScale(scale float64) {
	if scale <= 0 {
		scale = 1
	}
	if scale != t.scale {
		t.scale = scale
		t.faces = map[float64]*text.Face{}
	}
}

// px converts a DIP length to render pixels, rounding to the nearest pixel.
func (t *theme) px(dip float64) int { return int(dip*t.scale + 0.5) }

// face returns a cached ebitenui font face for a DIP size at the current scale.
func (t *theme) face(sizeDIP float64) *text.Face {
	if face, ok := t.faces[sizeDIP]; ok {
		return face
	}
	var face text.Face = &text.GoTextFace{Source: t.source, Size: sizeDIP * t.scale}
	t.faces[sizeDIP] = &face
	return &face
}

func (t *theme) insets(top, left, right, bottom float64) *widget.Insets {
	return &widget.Insets{Top: t.px(top), Left: t.px(left), Right: t.px(right), Bottom: t.px(bottom)}
}

func solid(c color.Color) *image.NineSlice { return image.NewNineSliceColor(c) }

func bordered(body, border color.Color, widthPx int) *image.NineSlice {
	return image.NewBorderedNineSliceColor(body, border, max(1, widthPx))
}

// buttonImages is the standard clickable look; border color varies by role.
func (t *theme) buttonImages(border color.RGBA) *widget.ButtonImage {
	width := t.px(1)
	return &widget.ButtonImage{
		Idle:     bordered(colorButtonIdle, border, width),
		Hover:    bordered(colorButtonHover, border, width),
		Pressed:  bordered(colorButtonDown, border, width),
		Disabled: bordered(colorRow, colorDisabled, width),
	}
}

func (t *theme) buttonText(idle color.RGBA) *widget.ButtonTextColor {
	return &widget.ButtonTextColor{Idle: idle, Hover: idle, Pressed: idle, Disabled: colorDisabled}
}

// button builds a labelled button that emits handler on click.
func (t *theme) button(label string, sizeDIP float64, border, textColor color.RGBA, handler func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(t.buttonImages(border)),
		widget.ButtonOpts.Text(label, t.face(sizeDIP), t.buttonText(textColor)),
		widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { handler() }),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
	)
}

// label builds static text.
func (t *theme) label(value string, sizeDIP float64, textColor color.Color) *widget.Text {
	return widget.NewText(widget.TextOpts.Text(value, t.face(sizeDIP), textColor))
}

// wrapped builds text that wraps at a DIP width.
func (t *theme) wrapped(value string, sizeDIP float64, textColor color.Color, widthDIP float64) *widget.Text {
	return widget.NewText(widget.TextOpts.Text(value, t.face(sizeDIP), textColor), widget.TextOpts.MaxWidth(float64(t.px(widthDIP))))
}

// column is a vertical row layout container with DIP spacing and padding.
func (t *theme) column(spacingDIP float64, padding *widget.Insets, background *image.NineSlice, opts ...widget.WidgetOpt) *widget.Container {
	containerOpts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(t.px(spacingDIP)),
			widget.RowLayoutOpts.Padding(padding),
		)),
		widget.ContainerOpts.WidgetOpts(opts...),
	}
	if background != nil {
		containerOpts = append(containerOpts, widget.ContainerOpts.BackgroundImage(background))
	}
	return widget.NewContainer(containerOpts...)
}

// rowOf is a horizontal row layout container.
func (t *theme) rowOf(spacingDIP float64, opts ...widget.WidgetOpt) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(t.px(spacingDIP)),
		)),
		widget.ContainerOpts.WidgetOpts(opts...),
	)
}

// stretch is the row-layout data that makes a child fill the cross axis.
func stretch() widget.WidgetOpt {
	return widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})
}
```

Check `widget.RowLayoutData` field names with `grep -n 'type RowLayoutData' -A 12 $(go env GOMODCACHE)/github.com/ebitenui/ebitenui@v0.7.3/widget/rowlayout.go`; it has `Position`, `Stretch`, `MaxWidth`, `MaxHeight`.

- [ ] **Step 6: Write `panel.go`**

```go
package hud

import (
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// Panel DIP geometry (spec §4). Map area: x 20–884, y 74–700.
const (
	panelX          = 908.0
	panelY          = 68.0
	panelWidth      = 352.0
	panelHeight     = 632.0
	panelPadding    = 18.0
	mapLeft         = 20.0
	mapRight        = 884.0
	mapBottom       = 700.0
	drawerCompactH  = 102.0
	drawerExpandedH = 300.0
	drawerTabW      = 150.0
	drawerTabH      = 18.0
)

// Panel owns the ebitenui tree for every piece of chrome. It rebuilds the tree
// whenever the State value changes and otherwise leaves widgets untouched so
// presses and hovers survive across ticks.
type Panel struct {
	ui       *ebitenui.UI
	theme    *theme
	root     *widget.Container
	last     State
	built    bool
	intents  []Intent
	handles  handles
}

// handles keeps pointers to widgets tests and refreshes need to reach.
type handles struct {
	endTurn   *widget.Button
	chips     map[uint32]*widget.Button // keyed by gameapi.BandID
	rowHeader [3]*widget.Button
	guideNext *widget.Button
	guideX    *widget.Button
	overlay   *widget.Window
}

func New() *Panel {
	panel := &Panel{theme: newTheme()}
	panel.root = widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}))
	panel.ui = &ebitenui.UI{Container: panel.root}
	return panel
}

// Update rebuilds on change, runs ebitenui, and returns the intents clicks
// produced this tick. Call it before map input so Hovered is current.
func (p *Panel) Update(state State) []Intent {
	if !p.built || state != p.last {
		p.rebuild(state)
		p.last = state
		p.built = true
	}
	p.ui.Update()
	intents := p.intents
	p.intents = nil
	return intents
}

func (p *Panel) Draw(screen *ebiten.Image) { p.ui.Draw(screen) }

// Hovered reports whether the pointer is over any chrome widget, so map input
// can yield. Valid after Update.
func (p *Panel) Hovered() bool { return input.UIHovered }

func (p *Panel) emit(intent Intent) { p.intents = append(p.intents, intent) }

// rect converts a DIP rectangle to render pixels including the letterbox offset.
func (p *Panel) rect(x, y, width, height float64) fixedRect {
	transform := p.last.Transform
	if transform.Scale <= 0 {
		transform.Scale = 1
	}
	left := int(transform.OffsetX + x*transform.Scale + 0.5)
	top := int(transform.OffsetY + y*transform.Scale + 0.5)
	return fixedRect(image.Rect(left, top, left+p.theme.px(width), top+p.theme.px(height)))
}

func (p *Panel) rebuild(state State) {
	p.last = state
	scale := state.Transform.Scale
	if scale <= 0 {
		scale = 1
	}
	p.theme.setScale(scale)
	if p.handles.overlay != nil {
		p.handles.overlay.Close()
		p.handles.overlay = nil
	}
	p.root.RemoveChildren()
	p.handles = handles{chips: map[uint32]*widget.Button{}}
	if state.Frame == nil {
		return
	}
	p.root.AddChild(p.buildPanel(state))
	if drawer := p.buildDrawer(state); drawer != nil {
		p.root.AddChild(drawer)
	}
	if ending := p.buildEndScene(state); ending != nil {
		p.root.AddChild(ending)
	}
	p.buildOverlay(state)
}
```

For this task, add temporary minimal versions of the four builders in `panel.go` so the package compiles; Tasks 8–13 replace them file by file:

```go
func (p *Panel) buildPanel(state State) widget.PreferredSizeLocateableWidget {
	column := p.theme.column(6, p.theme.insets(panelPadding, panelPadding, panelPadding, panelPadding), solid(colorPanel),
		widget.WidgetOpts.LayoutData(p.rect(panelX, panelY, panelWidth, panelHeight)))
	column.AddChild(p.theme.label("Africa 2 Ice", 24, colorTitle))
	return column
}
func (p *Panel) buildDrawer(State) widget.PreferredSizeLocateableWidget   { return nil }
func (p *Panel) buildEndScene(State) widget.PreferredSizeLocateableWidget { return nil }
func (p *Panel) buildOverlay(State)                                        {}
```

Remove `import _ "github.com/ebitenui/ebitenui"` from `doc.go`.

- [ ] **Step 7: Write the failing panel test**

Create `pkg/hud/panel_test.go`:

```go
package hud

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func testFrame(bands int) *gameapi.Frame {
	frame := &gameapi.Frame{CampaignResult: gameapi.Ongoing, YearBP: 76_400, Turn: 12,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 5, Y: 5, Land: true, Explored: true, Biome: gameapi.RiverineWoodland, BaselineK: 150, EcologicalK: 145, FloraStock: 212, FloraCap: 810, WaterStock: 315, WaterCap: 500, NaturalShelter: 0.5, MovementCost: 1.2},
			{ID: 1, X: 6, Y: 4, Land: true, Explored: true, Biome: gameapi.Savanna, BaselineK: 150, EcologicalK: 150, FloraStock: 640, FloraCap: 810, WaterStock: 480, WaterCap: 500, NaturalShelter: 0.4, MovementCost: 1.2},
		}}
	for index := 0; index < bands; index++ {
		band := gameapi.Band{ID: gameapi.BandID(index + 1), Species: gameapi.HomoSapiens, Population: 60, Health: 1, TileID: 0,
			AllocationBP:        [gameapi.AssignmentCount]uint16{3_500, 3_000, 1_500, 500, 1_500},
			MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1}}}
		band.ResearchOptions[gameapi.Firecraft] = gameapi.ResearchOption{Available: true, Cost: 80}
		frame.Bands = append(frame.Bands, band)
	}
	return frame
}

func testState(frame *gameapi.Frame, scale float64) State {
	viewport := render.NextViewport(render.Viewport{}, 1280*scale, 720*scale, 1)
	return State{
		Frame: frame, SelectedBand: 1, OpenRow: ui.RowMove, NotesMode: NotesCompact,
		Guide:    ui.NewGuideState(true),
		EndTurn:  ui.EndTurnGateFor(frame, false, false, false),
		Viewport: viewport, Transform: render.FitPresentation(viewport.RenderWidthPx, viewport.RenderHeightPx),
		Workforce: WorkforceDraft{Visible: true, Population: 60, AllocationBP: frame.Bands[0].AllocationBP, Valid: true},
	}
}

func TestPanelBuildsAtEveryScaleWithoutIntents(t *testing.T) {
	for _, scale := range []float64{1, 1.5, 2} {
		for _, bands := range []int{1, 9, 17, 256} {
			panel := New()
			state := testState(testFrame(bands), scale)
			if intents := panel.Update(state); len(intents) != 0 {
				t.Fatalf("scale %.1f bands %d: unsolicited intents %v", scale, bands, intents)
			}
			screen := ebiten.NewImage(int(1280*scale), int(720*scale))
			panel.Draw(screen)
			screen.Deallocate()
		}
	}
}

func TestPanelRebuildsOnlyWhenStateChanges(t *testing.T) {
	panel := New()
	state := testState(testFrame(2), 1)
	panel.Update(state)
	before := panel.root
	panel.Update(state)
	if panel.root != before || !panel.built {
		t.Fatal("panel root was replaced without a state change")
	}
	first := panel.root.Children()
	panel.Update(state)
	if len(panel.root.Children()) != len(first) || panel.root.Children()[0] != first[0] {
		t.Fatal("unchanged state rebuilt the widget tree")
	}
	state.OpenRow = ui.RowResearch
	panel.Update(state)
	if panel.root.Children()[0] == first[0] {
		t.Fatal("changed state did not rebuild the widget tree")
	}
}
```

Run: `go test ./pkg/hud/ -v` — Expected: PASS (the skeleton draws a title only; `Container.Children()` is exported in v0.7.3). Also add `builds int` to `Panel` and `p.builds++` in `rebuild` now; Task 9's test asserts on it.

- [ ] **Step 8: Verify boundaries and commit**

```bash
go build ./... && go vet ./... && go test ./pkg/hud/ ./internal/archtest/ && golangci-lint run
git add pkg/hud
git commit -m "Add the hud package skeleton: State, Intent, theme, fixed layout, Panel

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 8: Header, band chips, band line, and details section

**Files:**
- Create: `pkg/hud/header.go`, `pkg/hud/chips.go`, `pkg/hud/details.go`
- Modify: `pkg/hud/panel.go` (replace the temporary `buildPanel`), `pkg/hud/state.go` (add `BandListOpen bool` to `State`), `pkg/hud/intent.go` (add `IntentToggleBandList` before `IntentKindCount`)
- Test: `pkg/hud/panel_test.go`

**Interfaces:**
- Produces (package-private, used by Task 9):
  - `func (p *Panel) buildPanel(state State) widget.PreferredSizeLocateableWidget` — the full column: header, chips, band line, details, guide card (Task 16 fills), checklist (Task 9), end turn (Task 9), footer.
  - `func (p *Panel) buildChecklist(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget` — stub in this task returning an empty column; Task 9 implements.
  - `func (p *Panel) buildGuideCard(state State) widget.PreferredSizeLocateableWidget` — stub returning nil; Task 16 implements.
  - `handles.chips[uint32(bandID)]`, `handles.details *widget.Button`.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/hud/panel_test.go`:

```go
func TestChipsCarryProgressColorMarkerAndSelection(t *testing.T) {
	frame := testFrame(3)
	frame.Bands[1].HasQueuedMigration = true
	frame.Bands[2].Health = 0.4
	panel := New()
	panel.Update(testState(frame, 1))
	if len(panel.handles.chips) != 3 {
		t.Fatalf("chip count = %d", len(panel.handles.chips))
	}
	if got := panel.handles.chips[1].Text().Label; got != "B1" {
		t.Fatalf("selected chip label = %q", got)
	}
	if got := panel.handles.chips[3].Text().Label; got != "!! B3" {
		t.Fatalf("suffering chip label = %q, want the !! prefix", got)
	}
	panel.handles.chips[2].Click()
	intents := panel.Update(testState(frame, 1))
	if len(intents) != 1 || intents[0].Kind != IntentSelectBand || intents[0].Band != 2 {
		t.Fatalf("chip click intents = %+v", intents)
	}
}

func TestChipRowOverflowsIntoAPlusChip(t *testing.T) {
	panel := New()
	state := testState(testFrame(12), 1)
	panel.Update(state)
	if len(panel.handles.chips) != 7 {
		t.Fatalf("visible chips = %d, want 7 alongside the +N chip", len(panel.handles.chips))
	}
	if panel.handles.more == nil || panel.handles.more.Text().Label != "+5" {
		t.Fatal("overflow chip missing or mislabelled")
	}
	panel.handles.more.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentToggleBandList {
		t.Fatalf("+N click intents = %+v", intents)
	}
	state.BandListOpen = true
	panel.Update(state)
	if panel.handles.bandList == nil {
		t.Fatal("band list window not shown when BandListOpen")
	}
}

func TestDetailsDisclosureListsTraitsAsFocusButtons(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	panel.Update(state)
	if panel.handles.details == nil || len(panel.handles.traits) != 0 {
		t.Fatal("collapsed details should have a toggle and no trait cells")
	}
	panel.handles.details.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentToggleDetails {
		t.Fatalf("details click = %+v", intents)
	}
	state.DetailsOpen = true
	panel.Update(state)
	if len(panel.handles.traits) != int(gameapi.HeritableTraitCount) {
		t.Fatalf("trait cells = %d", len(panel.handles.traits))
	}
	panel.handles.traits[gameapi.PigmentationLevel].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentFocusTrait || intents[0].Trait != gameapi.PigmentationLevel {
		t.Fatalf("trait click = %+v", intents)
	}
}
```

Add to `handles`: `more *widget.Button`, `bandList *widget.Window`, `details *widget.Button`, `traits map[gameapi.HeritableTrait]*widget.Button` (initialise `traits` in `rebuild` alongside `chips`).

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run 'TestChip|TestDetails' -v` — Expected: FAIL (compile errors on missing handles/fields).

- [ ] **Step 3: Implement `header.go`**

```go
package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var eraRanges = [gameapi.CampaignEraCount]string{"80,000–50,000 BP", "50,000–35,000 BP", "35,000–25,000 BP", "25,000–20,000 BP"}

func macroWarning(frame *gameapi.Frame) string {
	for _, episode := range frame.MacroEpisodes {
		switch {
		case episode.Current:
			return "ACTIVE · " + episode.Episode.String()
		case episode.Warned:
			return "WARNING · " + episode.Episode.String()
		}
	}
	return ""
}

// buildHeader is the title row, clock, era/season/epoch, macro warning, and
// the sapiens total (spec §4 item 1).
func (p *Panel) buildHeader(state State) widget.PreferredSizeLocateableWidget {
	frame := state.Frame
	t := p.theme
	column := t.column(2, nil, nil, stretch())
	title := t.rowOf(8, stretch())
	title.AddChild(t.label("Africa 2 Ice", 24, colorTitle))
	menu := t.button("☰ Menu · Esc", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentOpenMenu}) })
	menu.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	title.AddChild(menu)
	column.AddChild(title)
	column.AddChild(t.label(fmt.Sprintf("%s BP · Turn %d/400", formatThousands(frame.YearBP), frame.Turn), 15, colorText))
	era := frame.Era.String() + " era"
	if frame.Era < gameapi.CampaignEraCount {
		era += " · " + eraRanges[frame.Era]
	}
	column.AddChild(t.label(era+" · "+frame.Season.String()+" · "+frame.Climate.Epoch.String(), 10.5, colorDim))
	if warning := macroWarning(frame); warning != "" {
		column.AddChild(t.label(warning, 9.5, colorAmber))
	}
	var total uint64
	living := 0
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 {
			total += uint64(band.Population)
			living++
		}
	}
	population := t.rowOf(6)
	population.AddChild(t.label(fmt.Sprintf("Homo sapiens %d", total), 14, colorGold))
	population.AddChild(t.label(fmt.Sprintf("· %d bands · Regions %d", living, len(frame.SapiensEstablishedRegions)), 10.5, colorDim))
	column.AddChild(population)
	return column
}

func formatThousands(value int) string {
	digits := fmt.Sprintf("%d", value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}
```

Check `widget.RowLayoutPositionEnd` exists (`grep -n RowLayoutPosition $(go env GOMODCACHE)/github.com/ebitenui/ebitenui@v0.7.3/widget/rowlayout.go`); if the constant is named differently, use the End variant it defines.

- [ ] **Step 4: Implement `chips.go`**

```go
package hud

import (
	"fmt"
	"image"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

const maxVisibleChips = 8

// visibleChipIDs returns at most eight band IDs in attention order, always
// including the selected band, and the count left over for the +N chip.
func visibleChipIDs(frame *gameapi.Frame, selected gameapi.BandID) ([]gameapi.BandID, int) {
	ordered := ui.SapiensBandIDsByAttention(frame.Bands)
	if len(ordered) <= maxVisibleChips {
		return ordered, 0
	}
	visible := append([]gameapi.BandID(nil), ordered[:maxVisibleChips-1]...)
	included := false
	for _, id := range visible {
		if id == selected {
			included = true
		}
	}
	if !included {
		for _, id := range ordered {
			if id == selected {
				visible[len(visible)-1] = id
			}
		}
	}
	return visible, len(ordered) - len(visible)
}

func (p *Panel) chipButton(band gameapi.Band, selected bool) *widget.Button {
	t := p.theme
	border, textColor := colorGold, colorGold
	if ui.MoveDone(band) {
		border, textColor = colorGreen, colorGreen
	}
	label := fmt.Sprintf("B%d", band.ID)
	if marker := ui.ConditionForSapiensBand(band).Marker(); marker != "" {
		label = marker + " " + label
	}
	images := t.buttonImages(border)
	if selected {
		images.Idle = bordered(colorButtonHover, colorText, t.px(2))
		images.Hover = images.Idle
	}
	id := band.ID
	button := widget.NewButton(
		widget.ButtonOpts.Image(images),
		widget.ButtonOpts.Text(label, t.face(10.5), t.buttonText(textColor)),
		widget.ButtonOpts.TextPadding(t.insets(1, 8, 8, 1)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentSelectBand, Band: id}) }),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
	)
	return button
}

// buildChips is the band row (spec §4 item 2): a 4-column grid so eight chips
// occupy two lines, plus a +N chip past eight bands.
func (p *Panel) buildChips(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(4), widget.GridLayoutOpts.Spacing(t.px(4), t.px(4)))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	visible, remaining := visibleChipIDs(state.Frame, state.SelectedBand)
	for _, id := range visible {
		for _, band := range state.Frame.Bands {
			if band.ID != id {
				continue
			}
			chip := p.chipButton(band, band.ID == state.SelectedBand)
			p.handles.chips[uint32(band.ID)] = chip
			grid.AddChild(chip)
		}
	}
	if remaining > 0 {
		more := t.button(fmt.Sprintf("+%d", remaining), 10.5, colorDim, colorText, func() { p.emit(Intent{Kind: IntentToggleBandList}) })
		p.handles.more = more
		grid.AddChild(more)
	}
	if state.BandListOpen {
		p.openBandList(state)
	}
	return grid
}

// openBandList shows every band in attention order as a modal list; clicking
// one selects it, and the application closes the list on selection.
func (p *Panel) openBandList(state State) {
	t := p.theme
	list := t.column(3, t.insets(12, 14, 14, 12), bordered(colorRowOpen, colorGoldDeep, t.px(1)))
	list.AddChild(t.label("ALL BANDS · priority order", 10, colorGoldDeep))
	for _, id := range ui.SapiensBandIDsByAttention(state.Frame.Bands) {
		for _, band := range state.Frame.Bands {
			if band.ID != id {
				continue
			}
			label := fmt.Sprintf("%-3s B%-3d pop %d · health %.0f%% · %s", ui.ConditionForSapiensBand(band).Marker(), band.ID, band.Population, band.Health*100, ui.MoveSummary(state.Frame, band))
			bandID := band.ID
			list.AddChild(t.button(label, 10, colorPanelEdge, colorText, func() { p.emit(Intent{Kind: IntentSelectBand, Band: bandID}) }))
		}
	}
	list.AddChild(t.button("Close · Esc", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleBandList}) }))
	rect := image.Rectangle(p.rect(panelX-340, panelY+60, 330, 40+16*float64(len(state.Frame.Bands))))
	window := widget.NewWindow(widget.WindowOpts.Contents(list), widget.WindowOpts.Modal(), widget.WindowOpts.CloseMode(widget.NONE), widget.WindowOpts.Location(rect))
	p.ui.AddWindow(window)
	p.handles.bandList = window
}
```

Close the list in `rebuild` the same way `overlay` is closed (add `if p.handles.bandList != nil { p.handles.bandList.Close() }` before resetting handles).

- [ ] **Step 5: Implement `details.go`**

```go
package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var traitShortNames = [gameapi.HeritableTraitCount]string{"Cold", "Altitude", "Immune", "Arid", "Pigment", "Fat"}

// buildBandLine is the one-line band identity with the details toggle
// (spec §4 item 3).
func (p *Panel) buildBandLine(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	row := t.rowOf(8, stretch())
	if band == nil {
		row.AddChild(t.label("No band selected", 13, colorText))
		return row
	}
	identity := fmt.Sprintf("Band %d", band.ID)
	detail := fmt.Sprintf("· Pop %d · Health %.0f%%", band.Population, band.Health*100)
	if band.Species == gameapi.ArchaicHominin {
		detail += " · Computer controlled · read only"
	}
	row.AddChild(t.label(identity, 13, colorText))
	row.AddChild(t.label(detail, 10.5, colorDim))
	label := "details ▾"
	if state.DetailsOpen {
		label = "details ▴"
	}
	toggle := t.button(label, 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleDetails}) })
	toggle.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	p.handles.details = toggle
	row.AddChild(toggle)
	return row
}

// buildDetails is the expanded disclosure (spec §4 item 4): food last turn,
// deaths last turn, heritable variants as focus buttons, interbreeding partners.
func (p *Panel) buildDetails(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(4, t.insets(6, 10, 10, 8), solid(colorRow), stretch())
	food := "Food last turn: unavailable"
	if report := band.LastFoodReport; report.Turn > 0 {
		food = fmt.Sprintf("Turn %d food: need %.1f · ate %.1f · short %.1f (%.0f%%)", report.Turn, report.RequiredFU, report.ConsumedFU(), report.DeficitFU, report.DeficitFraction()*100)
	}
	column.AddChild(t.label(food, 9.5, colorText))
	deaths := "Deaths last turn: unavailable"
	if band.LastOutcomeReport.Turn > 0 {
		m := band.LastMortality
		deaths = fmt.Sprintf("Deaths: starvation %.2f · seasonal %.2f · chronic %.2f · macro %.2f · acute %.2f", m.Starvation, m.Seasonal, m.Chronic, m.Macro, m.Acute)
	}
	column.AddChild(t.label(deaths, 9.5, colorText))
	column.AddChild(t.label(fmt.Sprintf("Stored food %.1f FU", band.StoredFood), 9.5, colorText))

	column.AddChild(t.label("HERITABLE VARIANTS · click one for its Field Note · G cycles", 8.5, colorDim))
	var tile gameapi.Tile
	if int(band.TileID) < len(state.Frame.Tiles) {
		tile = state.Frame.Tiles[band.TileID]
	}
	pressures := [gameapi.HeritableTraitCount]string{
		fmt.Sprintf("%.0f °C", tile.LocalTemperatureC), fmt.Sprintf("%.1f km", tile.ElevationKm), tile.Biome.String(),
		fmt.Sprintf("%.0f °C", tile.LocalTemperatureC), fmt.Sprintf("%.0f° lat", tile.Latitude), "diet",
	}
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(3), widget.GridLayoutOpts.Spacing(t.px(4), t.px(4)))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		focus := trait
		cell := t.button(fmt.Sprintf("%s %.3f @ %s", traitShortNames[trait], band.HeritableState[trait], pressures[trait]), 8.5, colorPanelEdge, colorText,
			func() { p.emit(Intent{Kind: IntentFocusTrait, Trait: focus}) })
		p.handles.traits[trait] = cell
		grid.AddChild(cell)
	}
	column.AddChild(grid)

	if len(band.InterbreedCandidateIDs) > 0 || band.HasInterbreedTarget {
		partners := "Interbreeding partners:"
		for _, candidate := range band.InterbreedCandidateIDs {
			partners += fmt.Sprintf(" B%d", candidate)
		}
		if band.HasInterbreedTarget {
			partners = fmt.Sprintf("Interbreeding accepted with B%d · gene flow resolves at end of turn", band.InterbreedTargetID)
		}
		column.AddChild(t.wrapped(partners, 9, colorInterbreed, panelWidth-2*panelPadding-20))
	}
	return column
}
```

- [ ] **Step 6: Replace `buildPanel` in `panel.go`**

```go
func (p *Panel) buildPanel(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(8, t.insets(14, panelPadding, panelPadding, 12), solid(colorPanel),
		widget.WidgetOpts.LayoutData(p.rect(panelX, panelY, panelWidth, panelHeight)))
	band := state.selectedBand()
	column.AddChild(p.buildHeader(state))
	column.AddChild(p.buildChips(state))
	column.AddChild(p.buildBandLine(state, band))
	if state.DetailsOpen && band != nil {
		column.AddChild(p.buildDetails(state, band))
	}
	if guide := p.buildGuideCard(state); guide != nil {
		column.AddChild(guide)
	}
	column.AddChild(p.buildChecklist(state, band))
	column.AddChild(t.label("Space ends the turn · Tab next band · ? shortcuts", 9.5, colorDim))
	return column
}

func (p *Panel) buildGuideCard(State) widget.PreferredSizeLocateableWidget { return nil }

func (p *Panel) buildChecklist(State, *gameapi.Band) widget.PreferredSizeLocateableWidget {
	return p.theme.column(6, nil, nil, stretch())
}
```

- [ ] **Step 7: Run tests and commit**

Run: `go test ./pkg/hud/ -v` — Expected: PASS.

```bash
git add pkg/hud
git commit -m "hud: header, band chips with progress color, band line, details disclosure

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 9: Checklist rows and the End turn button

**Files:**
- Create: `pkg/hud/move_row.go`, `pkg/hud/research_row.go`, `pkg/hud/workforce_row.go`, `pkg/hud/end_turn.go`
- Modify: `pkg/hud/panel.go` (`buildChecklist` real implementation; workforce refresh path)
- Test: `pkg/hud/panel_test.go`

**Interfaces:**
- Consumes: `ui.LiveabilityRows`, `ui.CurrentTileLiveability`, `ui.TargetTile`, `ui.TargetTileLiveability`, `ui.MoveSummary`, `ui.ResearchSummary`, `ui.WorkforceSummary`, `ui.RoleShortLabel`, `ui.EndTurnGate`.
- Produces: `handles.rowHeader[row]`, `handles.moveHere, best, split, interbreed *widget.Button`, `handles.research [gameapi.TechCount]*widget.Button`, `handles.workforce workforceHandles{sliders [5]*widget.Slider; values [5]*widget.Text; minus, plus [5]*widget.Button; total *widget.Text; apply, discard *widget.Button}`, `handles.endTurn`.
- Rebuild rule change: `Panel.Update` compares states with `Workforce` zeroed; a Workforce-only change calls `p.refreshWorkforce(state)` instead of rebuilding, so slider drags survive.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/hud/panel_test.go`:

```go
func TestRowHeadersOpenRowsAndMoveButtonsEmitIntents(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	if panel.handles.moveHere == nil || panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("Move here should be enabled while hovering a reachable tile")
	}
	panel.handles.moveHere.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentMoveTo || intents[0].Tile != 1 {
		t.Fatalf("Move here = %+v", intents)
	}
	panel.handles.best.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentMoveToBest {
		t.Fatalf("Best tile = %+v", intents)
	}
	if !panel.handles.interbreed.GetWidget().Disabled {
		t.Fatal("Interbreed enabled without a co-located archaic band")
	}
	panel.handles.rowHeader[ui.RowResearch].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentOpenRow || intents[0].Row != ui.RowResearch {
		t.Fatalf("row header = %+v", intents)
	}
	state.Hover = render.TileHover{}
	panel.Update(state)
	if !panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("Move here should be disabled with no target")
	}
}

func TestResearchRowListsAvailableTechnologiesAsButtons(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.OpenRow = ui.RowResearch
	panel.Update(state)
	if panel.handles.research[gameapi.Firecraft] == nil {
		t.Fatal("available technology has no button")
	}
	if panel.handles.research[gameapi.Campcraft] != nil && !panel.handles.research[gameapi.Campcraft].GetWidget().Disabled {
		t.Fatal("locked technology is clickable")
	}
	panel.handles.research[gameapi.Firecraft].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentChooseResearch || intents[0].Tech != gameapi.Firecraft {
		t.Fatalf("research click = %+v", intents)
	}
}

func TestWorkforceRowRefreshesWithoutRebuildingAndGuardsApply(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.OpenRow = ui.RowWorkforce
	panel.Update(state)
	builds := panel.builds
	sliders := panel.handles.workforce.sliders
	if sliders[0] == nil || !panel.handles.workforce.apply.GetWidget().Disabled {
		t.Fatal("clean draft should render sliders and a disabled Apply")
	}
	state.Workforce.AllocationBP[0] = 3_600
	state.Workforce.Dirty, state.Workforce.Valid = true, false
	panel.Update(state)
	if panel.builds != builds {
		t.Fatal("a workforce-only change rebuilt the tree and would break a slider drag")
	}
	if panel.handles.workforce.sliders[0] != sliders[0] || panel.handles.workforce.sliders[0].Current != 36 {
		t.Fatalf("slider not refreshed in place: %v", panel.handles.workforce.sliders[0].Current)
	}
	if !panel.handles.workforce.apply.GetWidget().Disabled || panel.handles.workforce.total.Label != "Total 101% · reduce 1% to apply" {
		t.Fatalf("invalid total not reflected: %q", panel.handles.workforce.total.Label)
	}
	panel.handles.workforce.plus[1].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentAdjustRole || intents[0].Role != gameapi.HuntingAndFishing || intents[0].Delta != 100 {
		t.Fatalf("plus click = %+v", intents)
	}
	state.Workforce.AllocationBP[1] = 2_900
	state.Workforce.Valid = true
	panel.Update(state)
	if panel.handles.workforce.apply.GetWidget().Disabled {
		t.Fatal("valid dirty draft left Apply disabled")
	}
	panel.handles.workforce.apply.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentApplyWorkforce {
		t.Fatalf("apply click = %+v", intents)
	}
}

func TestEndTurnButtonReflectsTheGate(t *testing.T) {
	panel := New()
	frame := testFrame(2)
	state := testState(frame, 1)
	panel.Update(state)
	if panel.handles.endTurn.GetWidget().Disabled || panel.handles.endTurn.Text().Label != "End turn · 2 bands still need a move" {
		t.Fatalf("soft gate button = %q", panel.handles.endTurn.Text().Label)
	}
	panel.handles.endTurn.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentEndTurn || intents[0].Force {
		t.Fatalf("soft click = %+v", intents)
	}
	state.EndTurn = ui.EndTurnGateFor(frame, true, false, false)
	panel.Update(state)
	if !panel.handles.endTurn.GetWidget().Disabled {
		t.Fatal("hard block did not disable the button")
	}
	state.EndTurn = ui.EndTurnGateFor(frame, false, false, true)
	panel.Update(state)
	panel.handles.endTurn.Click()
	if intents := panel.Update(state); len(intents) != 1 || !intents[0].Force {
		t.Fatalf("armed click = %+v", intents)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run 'TestRowHeaders|TestResearchRow|TestWorkforceRow|TestEndTurnButton' -v` — Expected: FAIL (missing handles).

- [ ] **Step 3: Implement `move_row.go`**

```go
package hud

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

func tierColor(tier ui.LiveabilityTier) color.RGBA {
	switch tier {
	case ui.TierAmber:
		return colorAmber
	case ui.TierRed:
		return colorRed
	default:
		return colorText
	}
}

func deltaMark(delta int) (string, color.RGBA) {
	switch {
	case delta > 0:
		return " ▲", colorGreen
	case delta < 0:
		return " ▼", colorRed
	default:
		return "", colorText
	}
}

// rowHeader is the always-visible line of a checklist row: number badge,
// title, and one-line summary. Clicking it opens the row.
func (p *Panel) rowHeader(row ui.ChecklistRow, done, open bool, summary string) *widget.Button {
	t := p.theme
	badge := colorPanelEdge
	if done {
		badge = colorGreen
	} else if row != ui.RowWorkforce {
		badge = colorGold
	}
	border := colorPanelEdge
	if open {
		border = colorGold
	}
	label := fmt.Sprintf("%d  %-10s %s", int(row)+1, row.Title(), summary)
	button := widget.NewButton(
		widget.ButtonOpts.Image(t.buttonImages(border)),
		widget.ButtonOpts.Text(label, t.face(11), t.buttonText(badge)),
		widget.ButtonOpts.TextPadding(t.insets(6, 10, 10, 6)),
		widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentOpenRow, Row: row}) }),
		widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
	)
	p.handles.rowHeader[row] = button
	return button
}

// buildMoveBody is the open Move row (spec §4.1).
func (p *Panel) buildMoveBody(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(4, t.insets(6, 24, 10, 8), solid(colorRowOpen), stretch())
	here := ui.CurrentTileLiveability(state.Frame, band)
	targetTile, source := ui.TargetTile(band, state.Preview, state.Hover)
	target := ui.TileLiveability{}
	if source != ui.TargetNone {
		target = ui.TargetTileLiveability(state.Frame, band, targetTile)
	}
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(3), widget.GridLayoutOpts.Spacing(t.px(8), t.px(2)), widget.GridLayoutOpts.Stretch([]bool{false, true, true}, nil))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	grid.AddChild(t.label("", 9.5, colorDim), t.label("HERE", 9.5, colorGold), t.label("TARGET · "+source.Label(), 9.5, colorCyan))
	for _, row := range ui.LiveabilityRows(band, here, target) {
		mark, markColor := deltaMark(row.Delta)
		grid.AddChild(t.label(row.Label, 9.5, colorDim))
		grid.AddChild(t.label(row.Here, 9.5, tierColor(row.HereTier)))
		targetLabel := t.label(row.Target+mark, 9.5, tierColor(row.TargetTier))
		if mark != "" && row.TargetTier == ui.TierNormal {
			targetLabel.SetColor(markColor)
		}
		grid.AddChild(targetLabel)
	}
	if !target.Available && source != ui.TargetNone {
		grid.AddChild(t.label("", 9, colorDim), t.label("", 9, colorDim), t.label(target.Status, 9, colorDim))
	}
	body.AddChild(grid)

	buttons := t.rowOf(6)
	done := ui.MoveDone(*band)
	canMove := !done && target.Reachable && source != ui.TargetQueued
	moveHere := t.button("Move here · Enter", 10.5, colorCyan, colorCyan, func() { p.emit(Intent{Kind: IntentMoveTo, Tile: targetTile}) })
	moveHere.GetWidget().Disabled = !canMove
	p.handles.moveHere = moveHere
	best := t.button("Best tile", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentMoveToBest}) })
	best.GetWidget().Disabled = done || len(band.MigrationCandidates) == 0
	p.handles.best = best
	split := t.button("Split · N", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSplit}) })
	split.GetWidget().Disabled = done
	p.handles.split = split
	partner := state.InterbreedFocus
	if partner == 0 && len(band.InterbreedCandidateIDs) > 0 {
		partner = band.InterbreedCandidateIDs[0]
	}
	interbreed := t.button("Interbreed · I", 10.5, colorInterbreed, colorInterbreed, func() { p.emit(Intent{Kind: IntentInterbreed, Band: partner}) })
	interbreed.GetWidget().Disabled = done || len(band.InterbreedCandidateIDs) == 0
	p.handles.interbreed = interbreed
	buttons.AddChild(moveHere, best, split, interbreed)
	body.AddChild(buttons)
	if len(band.InterbreedCandidateIDs) > 1 && !done {
		picker := t.rowOf(4)
		picker.AddChild(t.label("Partner:", 9, colorDim))
		for _, candidate := range band.InterbreedCandidateIDs {
			id := candidate
			border := colorPanelEdge
			if id == partner {
				border = colorInterbreed
			}
			picker.AddChild(t.button(fmt.Sprintf("B%d", id), 9, border, colorInterbreed, func() { p.emit(Intent{Kind: IntentInterbreed, Band: id}) }))
		}
		body.AddChild(picker)
	}
	hint := "Arrows move a cursor instead of the pointer · Esc clears it · staying put is fine"
	if band.Species == gameapi.ArchaicHominin {
		hint = "Computer controlled · no player actions"
	}
	body.AddChild(t.label(hint, 8.5, colorDim))
	return body
}
```

- [ ] **Step 4: Implement `research_row.go`**

```go
package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var techShortNames = [gameapi.TechCount]string{"Fire", "Haft", "Plants", "Clothes", "Cordage", "Camp", "Medicine", "Traps", "Navigation"}

func missingPrerequisites(option gameapi.ResearchOption, acquired uint16) string {
	label := ""
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		if option.PrerequisiteMask&^acquired&(1<<technology) == 0 {
			continue
		}
		if label != "" {
			label += " + "
		}
		label += techShortNames[technology]
	}
	return label
}

// buildResearchBody lists the nine technologies (spec §4.2); available ones
// are buttons, the rest explain their state.
func (p *Panel) buildResearchBody(_ State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(3, t.insets(6, 24, 10, 8), solid(colorRowOpen), stretch())
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		option := band.ResearchOptions[technology]
		progress := fmt.Sprintf("%.0f/%.0f", band.ResearchProgress[technology], option.Cost)
		label := fmt.Sprintf("%d  %-20s %s", int(technology)+1, technology.String(), progress)
		var status string
		border, textColor := colorPanelEdge, colorText
		switch {
		case option.Acquired:
			status, border, textColor = "learned", colorGreen, colorGreen
		case option.Current:
			status, border, textColor = "current", colorGold, colorGold
		case option.Available && band.Species == gameapi.HomoSapiens:
			status = "available"
		case band.Species == gameapi.ArchaicHominin:
			status, textColor = "computer", colorDim
		default:
			status, textColor = "needs "+missingPrerequisites(option, band.AcquiredTech), colorDim
		}
		tech := technology
		button := widget.NewButton(
			widget.ButtonOpts.Image(t.buttonImages(border)),
			widget.ButtonOpts.Text(label+" · "+status, t.face(9.5), t.buttonText(textColor)),
			widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentChooseResearch, Tech: tech}) }),
			widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
		)
		button.GetWidget().Disabled = option.Acquired || !option.Available || band.Species != gameapi.HomoSapiens
		p.handles.research[technology] = button
		body.AddChild(button)
	}
	body.AddChild(t.label("Up/Down highlight · Enter chooses · keys 1–9", 8.5, colorDim))
	return body
}
```

- [ ] **Step 5: Implement `workforce_row.go`**

```go
package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

type workforceHandles struct {
	sliders [gameapi.AssignmentCount]*widget.Slider
	values  [gameapi.AssignmentCount]*widget.Text
	minus   [gameapi.AssignmentCount]*widget.Button
	plus    [gameapi.AssignmentCount]*widget.Button
	total   *widget.Text
	apply   *widget.Button
	discard *widget.Button
	header  *widget.Button
}

func workforceTotalLabel(draft WorkforceDraft) string {
	var total int
	for _, points := range draft.AllocationBP {
		total += int(points)
	}
	percent := total / 100
	switch {
	case percent > 100:
		return fmt.Sprintf("Total %d%% · reduce %d%% to apply", percent, percent-100)
	case percent < 100:
		return fmt.Sprintf("Total %d%% · add %d%% to apply", percent, 100-percent)
	default:
		return "Total 100%"
	}
}

// buildWorkforceBody is the open Workforce row (spec §4.3). The widgets it
// creates are refreshed in place by refreshWorkforce so a drag survives.
func (p *Panel) buildWorkforceBody(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(3, t.insets(6, 24, 10, 8), solid(colorRowOpen), stretch())
	draft := state.Workforce
	for role := gameapi.WorkforceRole(0); role < gameapi.AssignmentCount; role++ {
		current := role
		line := t.rowOf(6, stretch())
		marker := "  "
		if role == draft.SelectedRole {
			marker = "› "
		}
		line.AddChild(t.label(fmt.Sprintf("%s%-14s", marker, ui.RoleShortLabel(role)), 9.5, colorText))
		minus := t.button("−", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: -100}) })
		p.handles.workforce.minus[role] = minus
		line.AddChild(minus)
		slider := widget.NewSlider(
			widget.SliderOpts.Direction(widget.DirectionHorizontal),
			widget.SliderOpts.MinMax(0, 100),
			widget.SliderOpts.InitialCurrent(int(draft.AllocationBP[role])/100),
			widget.SliderOpts.Images(&widget.SliderTrackImage{Idle: solid(colorPanelEdge), Hover: solid(colorPanelEdge)}, t.buttonImages(colorGoldDeep)),
			widget.SliderOpts.FixedHandleSize(t.px(10)),
			widget.SliderOpts.TrackOffset(0),
			widget.SliderOpts.PageSizeFunc(func() int { return 5 }),
			widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
				delta := args.Current*100 - int(p.last.Workforce.AllocationBP[current])
				if delta != 0 {
					p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: delta})
				}
			}),
			widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(120), t.px(12)), widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		)
		p.handles.workforce.sliders[role] = slider
		line.AddChild(slider)
		plus := t.button("+", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: 100}) })
		p.handles.workforce.plus[role] = plus
		line.AddChild(plus)
		value := t.label(fmt.Sprintf("%3d%%", int(draft.AllocationBP[role])/100), 9.5, colorText)
		p.handles.workforce.values[role] = value
		line.AddChild(value)
		body.AddChild(line)
	}
	total := t.label(workforceTotalLabel(draft), 9.5, colorText)
	p.handles.workforce.total = total
	body.AddChild(total)
	buttons := t.rowOf(6)
	apply := t.button("Apply · A / Enter", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentApplyWorkforce}) })
	discard := t.button("Discard · D", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentDiscardWorkforce}) })
	p.handles.workforce.apply, p.handles.workforce.discard = apply, discard
	buttons.AddChild(apply, discard)
	body.AddChild(buttons)
	body.AddChild(t.label("Up/Down pick a role · Left/Right or −/+ step 1% · Shift steps 5%", 8.5, colorDim))
	if band.Species == gameapi.ArchaicHominin {
		for role := range p.handles.workforce.sliders {
			p.handles.workforce.sliders[role].GetWidget().Disabled = true
			p.handles.workforce.minus[role].GetWidget().Disabled = true
			p.handles.workforce.plus[role].GetWidget().Disabled = true
		}
	}
	p.refreshWorkforce(state)
	return body
}

// refreshWorkforce updates the row's dynamic parts from the draft without
// recreating widgets.
func (p *Panel) refreshWorkforce(state State) {
	h := &p.handles.workforce
	draft := state.Workforce
	if h.total == nil {
		if h.header != nil {
			h.header.Text().Label = fmt.Sprintf("%d  %-10s %s", int(ui.RowWorkforce)+1, ui.RowWorkforce.Title(), ui.WorkforceSummary(draft.AllocationBP, draft.Dirty))
		}
		return
	}
	for role := range h.sliders {
		if h.sliders[role] == nil {
			continue
		}
		if percent := int(draft.AllocationBP[role]) / 100; h.sliders[role].Current != percent {
			h.sliders[role].Current = percent
		}
		h.values[role].Label = fmt.Sprintf("%3d%%", int(draft.AllocationBP[role])/100)
	}
	h.total.Label = workforceTotalLabel(draft)
	if draft.Valid {
		h.total.SetColor(colorText)
	} else {
		h.total.SetColor(colorRed)
	}
	h.apply.GetWidget().Disabled = !(draft.Dirty && draft.Valid)
	h.discard.GetWidget().Disabled = !draft.Dirty
	if h.header != nil {
		h.header.Text().Label = fmt.Sprintf("%d  %-10s %s", int(ui.RowWorkforce)+1, ui.RowWorkforce.Title(), ui.WorkforceSummary(draft.AllocationBP, draft.Dirty))
	}
}
```

Setting `slider.Current` directly is the documented way to move an ebitenui slider programmatically; if the version exposes `SetCurrent`, prefer it. Add `workforce workforceHandles` to `handles`.

- [ ] **Step 6: Implement `end_turn.go` and the real `buildChecklist`**

`pkg/hud/end_turn.go`:

```go
package hud

import "github.com/ebitenui/ebitenui/widget"

// buildEndTurn is the button whose label explains its own state (spec §5.3).
func (p *Panel) buildEndTurn(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	gate := state.EndTurn
	fill, textColor := colorGoldDeep, colorBlack
	if gate.Soft {
		fill = colorAmber
	}
	force := gate.Enabled && !gate.Soft && state.Frame != nil && ui.BandsNeedingMove(state.Frame.Bands) > 0
	button := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{Idle: solid(fill), Hover: solid(colorGold), Pressed: solid(colorGoldDeep), Disabled: bordered(colorRow, colorDisabled, t.px(1))}),
		widget.ButtonOpts.Text(gate.Label, t.face(13), &widget.ButtonTextColor{Idle: textColor, Hover: textColor, Pressed: textColor, Disabled: colorDisabled}),
		widget.ButtonOpts.TextPadding(t.insets(8, 12, 12, 8)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentEndTurn, Force: force}) }),
		widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
	)
	button.GetWidget().Disabled = !gate.Enabled
	p.handles.endTurn = button
	return button
}
```

Add `"github.com/adsouza/africa2ice/pkg/ui"` to its imports. In `panel.go` replace the stub `buildChecklist`:

```go
func (p *Panel) buildChecklist(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(4, nil, nil, stretch())
	column.AddChild(t.label("THIS TURN", 9.5, colorDim))
	if band == nil {
		column.AddChild(t.label("Select a band on the map or a chip above.", 10, colorText))
		column.AddChild(p.buildEndTurn(state))
		return column
	}
	summaries := [ui.ChecklistRowCount]string{
		ui.MoveSummary(state.Frame, *band), ui.ResearchSummary(*band), ui.WorkforceSummary(state.Workforce.AllocationBP, state.Workforce.Dirty),
	}
	dones := [ui.ChecklistRowCount]bool{ui.MoveDone(*band), ui.ResearchDone(*band), false}
	for row := ui.ChecklistRow(0); row < ui.ChecklistRowCount; row++ {
		open := row == state.OpenRow
		header := p.rowHeader(row, dones[row], open, summaries[row])
		if row == ui.RowWorkforce {
			p.handles.workforce.header = header
		}
		column.AddChild(header)
		if !open {
			continue
		}
		switch row {
		case ui.RowMove:
			column.AddChild(p.buildMoveBody(state, band))
		case ui.RowResearch:
			column.AddChild(p.buildResearchBody(state, band))
		case ui.RowWorkforce:
			if state.Workforce.Visible {
				column.AddChild(p.buildWorkforceBody(state, band))
			} else {
				column.AddChild(t.label("Computer controlled · allocation read only", 9.5, colorDim))
			}
		}
	}
	column.AddChild(p.buildEndTurn(state))
	return column
}
```

Add imports `"github.com/adsouza/africa2ice/pkg/gameapi"` and `"github.com/adsouza/africa2ice/pkg/ui"` to `panel.go`, and change `Update` so workforce-only changes refresh:

```go
func (p *Panel) Update(state State) []Intent {
	structural, lastStructural := state, p.last
	structural.Workforce, lastStructural.Workforce = WorkforceDraft{}, WorkforceDraft{}
	switch {
	case !p.built || structural != lastStructural:
		p.rebuild(state)
		p.built = true
	case state.Workforce != p.last.Workforce:
		p.last = state
		p.refreshWorkforce(state)
	}
	p.ui.Update()
	intents := p.intents
	p.intents = nil
	return intents
}
```

(`rebuild` already sets `p.last = state` and increments `p.builds`.)

- [ ] **Step 7: Run tests and commit**

Run: `go test ./pkg/hud/ -v` — Expected: PASS.

```bash
git add pkg/hud
git commit -m "hud: Move, Research, Workforce rows and the End turn button

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 10: Field Notes drawer

**Files:**
- Create: `pkg/hud/drawer.go`
- Modify: `pkg/hud/panel.go` (remove the `buildDrawer` stub)
- Test: `pkg/hud/panel_test.go`

**Interfaces:**
- Consumes: `render.FieldNote`, `State.NotesMode`, `State.Frame.Events`.
- Produces: `handles.drawerTab, drawerMore *widget.Button`, `handles.events []*widget.Button`.

- [ ] **Step 1: Write the failing tests**

```go
func TestDrawerHasThreeStatesAndClickableEvents(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 11, Kind: gameapi.EventMigration, Summary: "Band 1 migrated"}, {Turn: 12, Kind: gameapi.EventMigration, Summary: "Band 1 migrated again"}}
	panel := New()
	state := testState(frame, 1)
	state.Note = render.FieldNote{Topic: "FIRECRAFT", Introduction: "Controlled fire.", Context: "Attested early.", GameEffect: "Raises survival."}
	panel.Update(state)
	if panel.handles.drawerTab == nil || panel.handles.drawerTab.Text().Label != "hide notes · F" {
		t.Fatal("compact drawer tab missing")
	}
	if panel.handles.drawerMore == nil || panel.handles.drawerMore.Text().Label != "▴ more" {
		t.Fatal("compact drawer lacks the expand control")
	}
	if len(panel.handles.events) != 2 || panel.handles.events[0].Text().Label != "T12 · Migration · Band 1 migrated again" {
		t.Fatalf("event lines = %d, first %q", len(panel.handles.events), panel.handles.events[0].Text().Label)
	}
	panel.handles.events[0].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentFocusEvent || intents[0].Event != gameapi.EventMigration {
		t.Fatalf("event click = %+v", intents)
	}
	panel.handles.drawerMore.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentSetNotesMode || intents[0].Notes != NotesExpanded {
		t.Fatalf("more click = %+v", intents)
	}
	state.NotesMode = NotesExpanded
	panel.Update(state)
	if panel.handles.drawerMore.Text().Label != "▾ less" {
		t.Fatal("expanded drawer lacks the collapse control")
	}
	state.NotesMode = NotesHidden
	panel.Update(state)
	if panel.handles.drawerTab.Text().Label != "▴ notes · F  ·  T12 · Migration · Band 1 migrated again" {
		t.Fatalf("hidden tab = %q, want newest event retained", panel.handles.drawerTab.Text().Label)
	}
	if panel.handles.drawerMore != nil || len(panel.handles.events) != 0 {
		t.Fatal("hidden drawer still shows body controls")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run TestDrawer -v` — Expected: FAIL.

- [ ] **Step 3: Implement `drawer.go`**

```go
package hud

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/widget"
)

const drawerEventLines = 2

func noteBody(note render.FieldNote) string {
	parts := []string{"[color=9fb1ae]SUMMARY[/color] · " + note.Introduction}
	if note.Context != "" {
		parts = append(parts, "[color=9fb1ae]HISTORICAL CONTEXT[/color] · "+note.Context)
	}
	if note.GameEffect != "" {
		parts = append(parts, "[color=9fb1ae]GAME ABSTRACTION[/color] · "+note.GameEffect)
	}
	if note.Hint != "" {
		parts = append(parts, "[color=9fb1ae]HINT[/color] · "+note.Hint)
	}
	if note.References != "" {
		parts = append(parts, "[color=9fb1ae]SOURCES[/color] · "+note.References)
	}
	return strings.Join(parts, "\n")
}

func eventLine(event gameapi.Event) string {
	return fmt.Sprintf("T%d · %s · %s", event.Turn, event.Kind, event.Summary)
}

// newestEvents returns up to limit events, newest first.
func newestEvents(events []gameapi.Event, limit int) []gameapi.Event {
	result := make([]gameapi.Event, 0, limit)
	for index := len(events) - 1; index >= 0 && len(result) < limit; index-- {
		result = append(result, events[index])
	}
	return result
}

// drawerHeight is the drawer's DIP height for a mode; hidden is the tab only.
func drawerHeight(mode NotesMode) float64 {
	switch mode {
	case NotesCompact:
		return drawerCompactH
	case NotesExpanded:
		return drawerExpandedH
	default:
		return 0
	}
}

// buildDrawer places the Field Notes drawer over the bottom of the map
// (spec §4) with its edge tab, or just the tab when hidden.
func (p *Panel) buildDrawer(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	height := drawerHeight(state.NotesMode)
	root := widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height-drawerTabH, mapRight-mapLeft, height+drawerTabH))))
	tabRow := t.rowOf(4, widget.WidgetOpts.LayoutData(p.rect(mapRight-2*drawerTabW-8, mapBottom-height-drawerTabH, 2*drawerTabW, drawerTabH)))
	events := newestEvents(state.Frame.Events, drawerEventLines)
	if state.NotesMode == NotesHidden {
		label := "▴ notes · F"
		if len(events) > 0 {
			label += "  ·  " + eventLine(events[0])
		}
		tab := t.button(label, 9, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: NotesCompact}) })
		p.handles.drawerTab = tab
		tabRow.AddChild(tab)
		root.AddChild(tabRow)
		return root
	}
	moreLabel, moreMode := "▴ more", NotesExpanded
	if state.NotesMode == NotesExpanded {
		moreLabel, moreMode = "▾ less", NotesCompact
	}
	more := t.button(moreLabel, 9, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: moreMode}) })
	p.handles.drawerMore = more
	tab := t.button("hide notes · F", 9, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: NotesHidden}) })
	p.handles.drawerTab = tab
	tabRow.AddChild(more, tab)
	root.AddChild(tabRow)

	background := colorDrawer
	heading := "FIELD NOTES"
	headingColor := colorGoldDeep
	if state.Note.Topic != "" {
		heading += " · " + state.Note.Topic
	}
	if state.Note.Celebration {
		background, headingColor = colorCelebrate, colorGold
		heading = "BREAKTHROUGH · " + state.Note.Topic
	}
	body := t.column(4, t.insets(6, 12, 12, 6), solid(background),
		widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height, mapRight-mapLeft, height)))
	body.AddChild(t.label(heading, 10, headingColor))
	noteHeight := height - 24 - 14*float64(min(len(events), drawerEventLines)+1) - 12
	area := widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(mapRight-mapLeft-24), t.px(noteHeight)))),
		widget.TextAreaOpts.FontFace(t.face(9)),
		widget.TextAreaOpts.FontColor(colorText),
		widget.TextAreaOpts.ProcessBBCode(true),
		widget.TextAreaOpts.Text(noteBody(state.Note)),
	)
	body.AddChild(area)
	body.AddChild(t.label("RECENT EVENTS", 8, headingColor))
	if len(events) == 0 {
		body.AddChild(t.label("No campaign events yet.", 8.5, colorDim))
	}
	for _, event := range events {
		kind := event.Kind
		line := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{Idle: solid(background), Hover: solid(colorRowOpen), Pressed: solid(colorRow)}),
			widget.ButtonOpts.Text(eventLine(event), t.face(8.5), t.buttonText(colorDim)),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentFocusEvent, Event: kind}) }),
			widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
		)
		p.handles.events = append(p.handles.events, line)
		body.AddChild(line)
	}
	root.AddChild(body)
	return root
}
```

Add `drawerTab, drawerMore *widget.Button` and `events []*widget.Button` to `handles`; remove the `buildDrawer` stub from `panel.go`. The BBCode color tags need `ProcessBBCode(true)`; if the TextArea renders the tags literally, drop the tags and keep plain headings.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./pkg/hud/ -v` — Expected: PASS.

```bash
git add pkg/hud
git commit -m "hud: Field Notes drawer with hidden, compact, and expanded states

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 11: Wire the panel into the app and delete the old HUD

**Files:**
- Create: `pkg/app/hud_state.go`, `pkg/app/hud_intents.go`, `pkg/app/hud_test.go`
- Modify: `pkg/app/game.go` (struct fields, constructor, `Update`, `Draw`, `acceptCompletedTurn`, load completion, `startNewCampaign`, selection helpers)
- Modify: `pkg/render/map.go` (delete HUD drawing; new `Draw` signature; hovered-tile tint), `pkg/render/map_test.go`
- Delete: `pkg/render/band_window.go`, `band_window_test.go`, `tile_info.go`, `tile_info_test.go`, `outcome.go`, `outcome_test.go`
- Modify: `pkg/app/game_test.go`, `pkg/app/e2e_summary.go` if it referenced deleted symbols (check with `go build ./...`).

**Interfaces:**
- Produces:
  - `Game` fields: `panel *hud.Panel; openRow ui.ChecklistRow; rowChosen bool; detailsOpen, bandListOpen bool; endTurnArmed bool; guide ui.GuideState; notesMode hud.NotesMode`.
  - `func (g *Game) hudState() hud.State`
  - `func (g *Game) handleIntents(intents []hud.Intent)`
  - `func (g *Game) resetDisclosure()` — recompute `openRow` from `ui.DefaultOpenRow`, close details and band list, disarm End turn. Called from `ensureSelection` callers: selection change, load, new campaign, completed turn.
  - `func (g *Game) endTurn(force bool)` — the one End-turn path (Space and button).
  - `render.MapScene.Draw(screen *ebiten.Image, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, ending EndScene, resizeRequired bool)`.
  - `render.MapScene.SetTileHover` stays; `SetWorkforceDraft`, `SetMenuOverlay`, `SetFieldNoteScroll`, `SetInterbreedFocus` are removed. `render.WorkforceDraft`, `render.MenuOverlay`, `FieldNoteMaxScroll`, `fieldNoteLines`, `wrapTextLines`, `MenuOverlayRowAt`, `SettingsVolumeAt`, `FieldNotesToggleContains`, `FieldNotesPanelContains`, `NewCampaignButtonContains` are deleted. `render.FieldNote` stays (pkg/ui returns it).

- [ ] **Step 1: Delete the HUD from render and fix its tests**

In `pkg/render/map.go`: delete `drawHUD`, `drawControlsReference`, `drawTileInspector`, `drawWorkforceDraft`, `workforceDraftStatus`, `fieldNotePanelColors`, `drawResearchKeys`, `drawResearchDAG`, `researchNodeColors`, `missingPrerequisiteLabel`, `researchShortName`, `researchNodePoints`, `researchLegendEntries`, `drawSelectedBandInspector`, `drawMenuOverlay`, `MenuOverlayRowAt`, `SettingsVolumeAt`, `FieldNotesToggleContains`, `FieldNotesPanelContains`, `fieldNoteLines`, `FieldNoteMaxScroll`, `recentEventLines`, `wrapTextLines`, `truncateTextToWidth` (keep `wrapTextToWidth`/`wrapWords` for the notice), the `WorkforceDraft`, `MenuOverlay`, `TileHover`-unrelated HUD constants (`hudPanelX` … `bottomInspectorHeight`, `menuOverlay*`, `settingsSlider*`, `fieldNotes*`, `bandListOriginY`, `bandRowHeight`, `bandOutcomeOriginY`, `tileInspector*`, `interbreedPanelLineY`, `workforce*`, `controls*`, `visibleFieldNoteLines`, `eventLineFontSize`, `hudEventLineFontSize`, `fieldNoteWrapLimit`), and the `MapScene` fields `workforce`, `overlay`, `fieldNoteScroll`, `interbreedFocus` with their setters. Keep `campaignEraLabel` only if still referenced; otherwise delete it and its test. Remove `fieldNote`, `fieldNotesVisible`, `fieldNoteScroll`, `interbreedFocus`, `workforce`, `overlay` from `mapFrameKey`. In `drawFrame`, remove the calls to `drawHUD`, `drawResearchKeys`, `drawMenuOverlay`; keep terrain, overlays, bands, arrows, end scene, notice. Delete `pkg/render/end_scene.go`'s `NewCampaignButtonContains` and the button constants, and the button drawing inside `drawEndScene` (the panel draws the button in Task 13).

New `Draw` signature:

```go
func (scene *MapScene) Draw(screen *ebiten.Image, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, ending EndScene, resizeRequired bool)
```

Add the hovered-tile tint to `drawFrame` after `drawReachableTiles`:

```go
	if scene.hover.Visible && int(scene.hover.TileID) < len(frame.Tiles) && frame.Tiles[scene.hover.TileID].Explored {
		x, y := scene.tilePoint(frame.Tiles[scene.hover.TileID])
		vector.FillRect(screen, x-mapTileSize/2+0.7, y-mapTileSize/2+0.7, mapTileSize-1.8, mapTileSize-1.8, color.RGBA{R: 87, G: 211, B: 211, A: 70}, false)
	}
```

Delete `band_window.go`, `tile_info.go`, `outcome.go` and their tests; the map still needs `interbreedCandidateTiles` (keep `interbreed.go`, delete its `spatialControlHint`/`interbreedPanelLine`/`interbreedStatus` if unreferenced — `interbreedCandidateTiles` needs `interbreedStatus`, so keep those two). In `map_test.go`: delete `TestOverlayAndFieldNotesHitTargets`, `TestResearchLegendNamesEveryNodeColorState`, `TestFieldNoteWrappingPreservesWordsAndBoundsLines`, `TestFieldNoteMaxScrollUsesRenderedLineLayout`, `TestFieldNoteWrapUsesAvailablePanelWidth`, `TestCleanWorkforceDraftHasNoPersistentStatus`, `TestRecentEventLinesShowNewestFirstAndStayBounded`, `TestRecentEventLinesTruncateByMeasuredWidthNotRuneCount`; update `renderMapOffscreen` to the new signature and rename `TestMapSceneDrawsFieldNotesAndTerminalVariantsOffscreen` to `TestMapSceneDrawsTerminalVariantsOffscreen`, keeping only the end-scene subtests. Update the tests in `tile_info_test.go` that covered `targetTileSummary` — that behaviour is now covered by `pkg/ui/liveability_test.go`.

Run: `go build ./pkg/render/ && go test ./pkg/render/` — Expected: PASS. (`pkg/app` will not compile yet.)

- [ ] **Step 2: Write the failing app tests**

Create `pkg/app/hud_test.go`:

```go
package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestHUDStateDerivesRowsAndGateFromTheFrame(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].HasQueuedMigration, frame.Bands[0].QueuedMigration = true, 2
	game := New(&gameStub{frame: frame})
	state := game.hudState()
	if state.SelectedBand != 7 || state.OpenRow != ui.RowResearch {
		t.Fatalf("loaded queued migration: selected %d open row %v, want band 7 with Research open", state.SelectedBand, state.OpenRow)
	}
	if !state.EndTurn.Enabled || state.EndTurn.Soft {
		t.Fatalf("gate with every band moved = %+v", state.EndTurn)
	}
	if state.NotesMode != hud.NotesCompact {
		t.Fatalf("default notes mode = %v", state.NotesMode)
	}
}

func TestEndTurnIntentHonorsTheDirtyDraftGuardLikeSpace(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.editAssignmentDraft(100)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn}})
	if stub.endTurns != 0 || game.notice != "Apply or discard workforce changes before ending the turn" {
		t.Fatalf("dirty-draft end turn = %d turns, notice %q", stub.endTurns, game.notice)
	}
	game.discardAssignmentDraft()
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn}})
	if stub.endTurns != 0 || !game.endTurnArmed {
		t.Fatalf("soft block: turns %d armed %t", stub.endTurns, game.endTurnArmed)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn, Force: true}})
	if stub.endTurns != 1 || game.endTurnArmed {
		t.Fatalf("armed click: turns %d armed %t", stub.endTurns, game.endTurnArmed)
	}
}

func TestIntentsReuseHotkeyPaths(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}})
	if command, ok := stub.appliedCommand.(gameapi.QueueMigration); !ok || command.TileID != 2 {
		t.Fatalf("MoveTo applied %#v", stub.appliedCommand)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentChooseResearch, Tech: gameapi.Firecraft}})
	if command, ok := stub.appliedCommand.(gameapi.ResearchTech); !ok || command.Tech != gameapi.Firecraft {
		t.Fatalf("ChooseResearch applied %#v", stub.appliedCommand)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentAdjustRole, Role: gameapi.Toolcraft, Delta: 100}})
	if game.assignmentRole != gameapi.Toolcraft || !game.assignmentDraftDirty() {
		t.Fatalf("AdjustRole: role %v dirty %t", game.assignmentRole, game.assignmentDraftDirty())
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentDiscardWorkforce}})
	game.handleIntents([]hud.Intent{{Kind: hud.IntentOpenRow, Row: ui.RowWorkforce}, {Kind: hud.IntentToggleDetails}})
	if game.openRow != ui.RowWorkforce || !game.detailsOpen {
		t.Fatal("OpenRow/ToggleDetails not applied")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSetNotesMode, Notes: hud.NotesExpanded}})
	if game.notesMode != hud.NotesExpanded || !game.settings.FieldNotesExpanded || !game.settings.FieldNotesVisible {
		t.Fatalf("SetNotesMode persisted %+v", game.settings)
	}
}

func TestSelectionChangeResetsDisclosure(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 50, TileID: 0, HasResearchTarget: true, SpatialActionUsed: true})
	game := New(&gameStub{frame: frame})
	game.detailsOpen, game.openRow, game.rowChosen = true, ui.RowWorkforce, true
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 9}})
	if game.selectedBand != 9 || game.detailsOpen || game.openRow != ui.RowMove || game.rowChosen {
		t.Fatalf("after select: band %d details %t row %v chosen %t", game.selectedBand, game.detailsOpen, game.openRow, game.rowChosen)
	}
}
```

Run: `go test ./pkg/app/ -run 'TestHUDState|TestEndTurnIntent|TestIntentsReuse|TestSelectionChangeResets'` — Expected: FAIL (does not compile).

- [ ] **Step 3: Add fields and constructor wiring in `game.go`**

Add to `Game`:

```go
	panel        *hud.Panel
	openRow      ui.ChecklistRow
	rowChosen    bool // player opened a row explicitly; auto-advance yields until reset
	detailsOpen  bool
	bandListOpen bool
	endTurnArmed bool
	guide        ui.GuideState
	notesMode    hud.NotesMode
```

Remove `fieldNoteScroll` and `fieldNotesVisible` (visibility now derives from `notesMode`; keep `settings.FieldNotesVisible` as the persisted source). In `newGameWithPresentation` set `panel: hud.New(), notesMode: hud.NotesCompact, guide: ui.NewGuideState(false)` and after `game.syncAssignmentDraft(true)` call `game.resetDisclosure()`. In `updateUISettings` (which installs loaded/changed settings) derive:

```go
	g.notesMode = notesModeFor(settings)
	g.guide = ui.NewGuideState(settings.GuideDismissed)
```

only when the settings came from a completed read (the first install); on later writes keep `g.guide` as is. Add:

```go
func notesModeFor(settings ui.UISettings) hud.NotesMode {
	switch {
	case !settings.FieldNotesVisible:
		return hud.NotesHidden
	case settings.FieldNotesExpanded:
		return hud.NotesExpanded
	default:
		return hud.NotesCompact
	}
}
```

Replace every read of `g.fieldNotesVisible` with `g.notesMode != hud.NotesHidden`; `toggleFieldNotes` flips `settings.FieldNotesVisible` and, on show, keeps `FieldNotesExpanded`. Add `setNotesMode(mode hud.NotesMode)` that writes `FieldNotesVisible = mode != NotesHidden` and `FieldNotesExpanded = mode == NotesExpanded` through `updateUISettings`.

- [ ] **Step 4: Write `hud_state.go`**

```go
package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// hudState derives everything the chrome draws from the accepted frame and
// UI-local fields. Nothing here is stored.
func (g *Game) hudState() hud.State {
	frame := g.displayFrame()
	note := g.fieldNote
	note.Celebration = g.breakthroughFrames > 0
	state := hud.State{
		Frame:        frame,
		SelectedBand: g.selectedBand,
		Preview:      render.MigrationPreview{BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview},
		Hover:        render.TileHover{TileID: g.hoveredTile, Visible: g.hasHoveredTile},
		OpenRow:      g.openRow,
		DetailsOpen:  g.detailsOpen,
		BandListOpen: g.bandListOpen,
		Workforce: hud.WorkforceDraft{
			Visible: g.hasAssignmentDraft, AllocationBP: g.assignmentDraft, SelectedRole: g.assignmentRole,
			Dirty: g.assignmentDraftDirty(), Valid: g.assignmentDraftValid(),
		},
		InterbreedFocus: g.interbreedFocus,
		EndTurn:         ui.EndTurnGateFor(frame, g.assignmentDraftDirty(), g.hasMigrationPreview, g.endTurnArmed),
		Note:            note,
		NotesMode:       g.notesMode,
		Guide:           g.guide,
		Ending:          ui.CampaignEndScene(frame),
		Viewport:        g.viewport,
		Transform:       render.FitPresentation(g.viewport.RenderWidthPx, g.viewport.RenderHeightPx),
	}
	if band := g.selected(); band != nil {
		state.Workforce.Population = band.Population
	}
	if frame != nil && frame.CampaignResult != gameapi.Ongoing {
		state.EndTurn = ui.EndTurnGate{}
	}
	state.Overlay = g.overlayState()
	return state
}

// resetDisclosure returns the panel to its defaults for a new selection, a
// load, a new campaign, or a completed turn (spec §5.2).
func (g *Game) resetDisclosure() {
	g.openRow = ui.DefaultOpenRow(g.selected())
	g.rowChosen = false
	g.detailsOpen = false
	g.bandListOpen = false
	g.endTurnArmed = false
}

// advanceOpenRow moves to the next unfinished row after an accepted action
// unless the player chose a row explicitly.
func (g *Game) advanceOpenRow() {
	if !g.rowChosen {
		g.openRow = ui.DefaultOpenRow(g.selected())
	}
}
```

`overlayState()` is added in Task 13; for now add a stub `func (g *Game) overlayState() hud.OverlayState { return hud.OverlayState{Scene: g.scenes.Current()} }` in `hud_state.go`. `render.FitPresentation(0, 0)` returns `Scale: 1` with zero offsets, so an uninitialised viewport (tests) needs no guard.

- [ ] **Step 5: Write `hud_intents.go`**

```go
package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// handleIntents maps chrome clicks onto the same guarded methods hotkeys use.
func (g *Game) handleIntents(intents []hud.Intent) {
	for _, intent := range intents {
		g.handleIntent(intent)
	}
}

func (g *Game) handleIntent(intent hud.Intent) {
	switch intent.Kind {
	case hud.IntentSelectBand:
		g.selectBandByID(intent.Band)
		g.bandListOpen = false
	case hud.IntentToggleBandList:
		g.bandListOpen = !g.bandListOpen
	case hud.IntentOpenRow:
		g.openRow, g.rowChosen = intent.Row, true
	case hud.IntentToggleDetails:
		g.detailsOpen = !g.detailsOpen
	case hud.IntentSetNotesMode:
		g.setNotesMode(intent.Notes)
	case hud.IntentMoveTo:
		if band := g.selected(); band != nil {
			g.clearMigrationPreview()
			if g.tryQueueMigration(band, intent.Tile) {
				g.advanceOpenRow()
			}
		}
	case hud.IntentMoveToBest:
		if band := g.selected(); band != nil {
			for _, candidate := range band.MigrationCandidates {
				if !candidate.RequiresPassage {
					g.clearMigrationPreview()
					if g.tryQueueMigration(band, candidate.TileID) {
						g.advanceOpenRow()
					}
					return
				}
			}
			g.showNotice("No reachable land tile to move to this turn.")
		}
	case hud.IntentSplit:
		g.splitSelectedBand()
		g.advanceOpenRow()
	case hud.IntentInterbreed:
		if intent.Band != 0 {
			g.interbreedFocus = intent.Band
		}
		g.requestInterbreed()
		g.advanceOpenRow()
	case hud.IntentChooseResearch:
		g.chooseResearchTechnology(intent.Tech)
		g.advanceOpenRow()
	case hud.IntentSelectRole:
		g.assignmentRole = intent.Role
	case hud.IntentAdjustRole:
		g.assignmentRole = intent.Role
		g.editAssignmentDraft(intent.Delta)
	case hud.IntentApplyWorkforce:
		g.applyAssignmentDraft()
	case hud.IntentDiscardWorkforce:
		g.discardAssignmentDraft()
	case hud.IntentEndTurn:
		g.endTurn(intent.Force)
	case hud.IntentGuideNext:
		g.guide = g.guide.Next()
	case hud.IntentGuideDismiss:
		g.dismissGuide()
	case hud.IntentFocusTrait:
		if band := g.selected(); band != nil {
			if note, ok := ui.TraitFieldNote(intent.Trait, band.HeritableState[intent.Trait]); ok {
				g.traitFocus = intent.Trait
				g.setFieldNote(note)
			}
		}
	case hud.IntentFocusEvent:
		if note, ok := ui.EventKindFieldNote(intent.Event); ok {
			g.setFieldNote(note)
		}
	case hud.IntentOpenMenu:
		g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneMenu)})
	default:
		g.handleOverlayIntent(intent)
	}
}

// selectBandByID is the chip/list selection path; it applies the dirty-draft
// guard exactly like Tab.
func (g *Game) selectBandByID(id gameapi.BandID) {
	if id == g.selectedBand {
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes")
		return
	}
	for _, band := range g.frame.Bands {
		if band.ID != id {
			continue
		}
		g.selectedBand = id
		g.clearMigrationPreview()
		g.syncAssignmentDraft(true)
		g.refreshBandFieldNote()
		g.resetDisclosure()
		return
	}
}

// endTurn is the single End-turn path for Space and the button (spec §5.3).
func (g *Game) endTurn(force bool) {
	if g.frame == nil || g.frame.CampaignResult != gameapi.Ongoing {
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes before ending the turn")
		return
	}
	if g.hasMigrationPreview {
		g.showNotice("Press Enter to queue the migration, or Esc to clear it before ending the turn.")
		return
	}
	if waiting := ui.BandsNeedingMove(g.frame.Bands); waiting > 0 && !force && !g.endTurnArmed {
		g.endTurnArmed = true
		g.showNotice("Some bands have not moved. Click End turn again or press Space to end the turn anyway.")
		return
	}
	g.endTurnArmed = false
	g.dispatchBatch([]ui.Action{ui.EndTurnAction()})
}

func (g *Game) dismissGuide() {
	g.guide = g.guide.Dismiss()
	if g.settingsLoading {
		return
	}
	settings := g.settings
	settings.GuideDismissed = true
	g.updateUISettings(settings)
}
```

Add `IntentToggleBandList` to the `hud` intent kinds if Task 8 has not (it has). `handleOverlayIntent` is a stub until Task 13: `func (g *Game) handleOverlayIntent(hud.Intent) {}` in `hud_intents.go`.

- [ ] **Step 6: Rewire `Update` and `Draw`**

In `Game.Draw`, replace the body's scene calls with:

```go
	g.scene.SetTileHover(render.TileHover{TileID: g.hoveredTile, Visible: g.hasHoveredTile})
	displayFrame := g.displayFrame()
	g.scene.Draw(screen, displayFrame, g.selectedBand, render.MigrationPreview{
		BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview,
	}, g.notice, ui.CampaignEndScene(displayFrame), g.viewportInitialized && !g.viewport.SupportsGameplay())
	g.panel.Draw(screen)
```

In `Game.Update`, after the `startupRestorePending`/`frame == nil`/viewport guards and before the quick-save shortcut, insert:

```go
	g.handleIntents(g.panel.Update(g.hudState()))
```

Then in the gameplay branch, gate map input on hover: replace the `IsMouseButtonJustPressed` block that called `FieldNotesToggleContains`/`handleMapClick` with:

```go
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.panel.Hovered() {
		g.handleMapClick()
	}
```

and in `syncTileHover` return early with `hasHoveredTile = false` when `g.panel.Hovered()`. Replace the Space handler body with `g.endTurn(false)`. Remove the `fieldNotesVisible` wheel/PageUp/PageDown block (Task 12 rebinds those keys). Delete `handleGameplayHotkey`'s F case if it referenced deleted render functions; `toggleFieldNotes` now calls `setNotesMode`.

Call `g.resetDisclosure()` at the end of `acceptCompletedTurn` (after `syncAssignmentDraft`), in `startNewCampaign`, in the `ReplacementFrame` load branch of `pollStorage`, and in `selectSapiens`/`selectBandAtTile` after `refreshBandFieldNote()`. In `acceptCompletedTurn` also `g.guide = g.guide.ObserveTurnCompleted()`, and after every successful `apply` in `dispatchBatch`'s `ActionSimulationCommand` case add `g.guide = g.guide.Observe(g.selected())`.

Delete `menuOverlayForRender`, `workforceDraftForRender`, `scrollFieldNotes`, and the `fieldNoteScroll` references; `TestFieldNoteScrollNormalizesBeforeApplyingInput` is deleted (the TextArea owns scrolling). Update `TestTitleAndGameMenuExposeCampaignNavigation`, `TestGameMenuDescribesTurnBasedBehavior`, `TestStorageBrowser*`, `TestSettingsSceneReportsLivePreferences` to read `game.overlayState()` in Task 13; for now mark them with `t.Skip("rewritten in Task 13")` so the package compiles, and remove the skips in Task 13. `TestCompletedTurnCelebratesNewTechnologyWithoutOverridingPanelVisibility` sets `game.fieldNotesVisible = false`; change to `game.notesMode = hud.NotesHidden` and assert `game.notesMode == hud.NotesHidden` afterwards.

- [ ] **Step 7: Build, test, commit**

Run: `go build ./... && go vet ./... && go test ./... && golangci-lint run` — Expected: PASS with the four skipped tests. Then:

```bash
go run . -screenshot /tmp/hud-task11.png -turns 0
```

Open the PNG and confirm: no bottom strip, the map is taller, the panel shows header, chips, band line, THIS TURN rows with Move open, End turn, and the drawer sits over the map's bottom edge.

```bash
git add -A pkg/render pkg/app
git commit -m "Wire the hud panel into the app and delete the old HUD drawing

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 12: Keyboard routing — row-owned keys, removed keys, Esc layering

**Files:**
- Create: `pkg/app/input.go`, `pkg/app/input_test.go`
- Modify: `pkg/app/game.go` (`Update` gameplay branch delegates to `handleGameplayKeys`; `handleSceneInput` gameplay Esc case)

**Interfaces:**
- Produces: `func (g *Game) handleGameplayKeys()`; `func (g *Game) handleRowKeys(row ui.ChecklistRow)`; `func (g *Game) changeOpenRow(delta int)`; `func (g *Game) escape() bool` (returns true when a layer was peeled); `func (g *Game) toggleShortcutSheet()` with field `shortcutsOpen bool` (drawn by the panel as a modal window listing spec §8 keys; add `ShortcutsOpen bool` to `hud.State` and `IntentToggleShortcuts` handled by setting the field). `researchCursor gameapi.Tech` field for Up/Down in the Research row.

- [ ] **Step 1: Write the failing tests**

Because Ebitengine key state cannot be injected in tests, the routing functions take the decoded key as a parameter. Create `pkg/app/input_test.go`:

```go
package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestArrowsBelongToTheOpenRow(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.openRow = ui.RowMove
	game.handleRowKey(ebiten.KeyArrowLeft, false)
	if !game.hasMigrationPreview {
		t.Fatal("Left in the Move row did not move the destination cursor")
	}
	game.clearMigrationPreview()

	game.openRow, game.rowChosen = ui.RowWorkforce, true
	game.assignmentRole = gameapi.Foraging
	game.handleRowKey(ebiten.KeyArrowDown, false)
	if game.assignmentRole != gameapi.HuntingAndFishing {
		t.Fatalf("Down in Workforce selected %v", game.assignmentRole)
	}
	game.handleRowKey(ebiten.KeyArrowRight, false)
	game.handleRowKey(ebiten.KeyArrowRight, true)
	if got := game.assignmentDraft[gameapi.HuntingAndFishing]; got != 600 {
		t.Fatalf("Right then Shift+Right = %d BP, want 600", got)
	}
	if game.hasMigrationPreview {
		t.Fatal("Workforce arrows leaked into the migration cursor")
	}
	game.handleRowKey(ebiten.KeyEnter, false)
	if stub.appliedCommand != nil {
		t.Fatal("Enter applied an invalid (>100%) draft")
	}
	game.handleRowKey(ebiten.KeyArrowLeft, true)
	game.handleRowKey(ebiten.KeyArrowLeft, false)
	game.handleRowKey(ebiten.KeyEnter, false)
	if _, ok := stub.appliedCommand.(gameapi.SetAssignment); ok {
		t.Fatal("Enter applied a draft equal to the baseline (nothing to apply)")
	}

	game.openRow = ui.RowResearch
	game.researchCursor = gameapi.Firecraft
	game.handleRowKey(ebiten.KeyArrowDown, false)
	game.handleRowKey(ebiten.KeyEnter, false)
	if command, ok := stub.appliedCommand.(gameapi.ResearchTech); !ok || command.Tech != gameapi.HaftedTools {
		t.Fatalf("Research Down+Enter applied %#v", stub.appliedCommand)
	}
}

func TestPageKeysAndShiftArrowsChangeTheOpenRow(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.openRow = ui.RowMove
	game.changeOpenRow(1)
	game.changeOpenRow(1)
	if game.openRow != ui.RowWorkforce || !game.rowChosen {
		t.Fatalf("two steps down = %v chosen %t", game.openRow, game.rowChosen)
	}
	game.changeOpenRow(1)
	if game.openRow != ui.RowMove {
		t.Fatal("open row did not wrap")
	}
	game.changeOpenRow(-1)
	if game.openRow != ui.RowWorkforce {
		t.Fatal("open row did not wrap backwards")
	}
}

func TestEscapePeelsOneLayerAtATime(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.handleDirectionalMigration(-1, -1)
	game.bandListOpen = true
	if !game.escape() || game.hasMigrationPreview || !game.bandListOpen {
		t.Fatal("first Esc should clear only the cursor")
	}
	if !game.escape() || game.bandListOpen {
		t.Fatal("second Esc should close the band list")
	}
	if game.escape() {
		t.Fatal("third Esc has nothing to peel and should return false so the menu opens")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/app/ -run 'TestArrowsBelong|TestPageKeys|TestEscapePeels'` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement `pkg/app/input.go`**

```go
package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// handleGameplayKeys is the keyboard half of spec §8: global keys first, then
// the keys the open checklist row owns.
func (g *Game) handleGameplayKeys() {
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyTab):
		if g.assignmentDraftDirty() {
			g.showNotice("Apply or discard workforce changes")
		} else {
			g.clearMigrationPreview()
			if shift {
				g.selectPreviousSapiens()
			} else {
				g.selectNextSapiens()
			}
		}
	case inpututil.IsKeyJustPressed(fieldNotesHotkey):
		if shift {
			g.toggleNotesExpanded()
		} else {
			g.toggleFieldNotes()
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyM):
		g.toggleMute()
	case inpututil.IsKeyJustPressed(ebiten.KeyZ):
		g.toggleCameraOverride()
	case inpututil.IsKeyJustPressed(ebiten.KeySlash) && shift:
		g.shortcutsOpen = !g.shortcutsOpen
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		g.changeOpenRow(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		g.changeOpenRow(1)
	case shift && inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
		g.changeOpenRow(-1)
	case shift && inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
		g.changeOpenRow(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyA):
		g.applyAssignmentDraft()
	case inpututil.IsKeyJustPressed(ebiten.KeyD):
		g.discardAssignmentDraft()
	case inpututil.IsKeyJustPressed(splitBandHotkey):
		g.splitSelectedBand()
		g.advanceOpenRow()
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		g.requestInterbreed()
		g.advanceOpenRow()
	case inpututil.IsKeyJustPressed(ebiten.KeyJ):
		g.selectNextInterbreedTarget()
	case inpututil.IsKeyJustPressed(ebiten.KeyG):
		g.focusNextTraitNote()
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.endTurn(false)
	case inpututil.IsKeyJustPressed(ebiten.KeyMinus):
		g.adjustSelectedRole(-100, shift)
	case inpututil.IsKeyJustPressed(ebiten.KeyEqual):
		g.adjustSelectedRole(100, shift)
	default:
		for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if inpututil.IsKeyJustPressed(key) {
				g.chooseResearchTechnology(gameapi.Tech(index))
				g.advanceOpenRow()
				return
			}
		}
		for _, key := range [...]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyEnter} {
			if inpututil.IsKeyJustPressed(key) {
				g.handleRowKey(key, shift)
				return
			}
		}
	}
}

// handleRowKey routes arrows and Enter to whichever row is open.
func (g *Game) handleRowKey(key ebiten.Key, shift bool) {
	switch g.openRow {
	case ui.RowMove:
		switch key {
		case ebiten.KeyArrowUp:
			g.handleDirectionalMigration(0, -1)
		case ebiten.KeyArrowDown:
			g.handleDirectionalMigration(0, 1)
		case ebiten.KeyArrowLeft:
			g.handleDirectionalMigration(-1, 0)
		case ebiten.KeyArrowRight:
			g.handleDirectionalMigration(1, 0)
		case ebiten.KeyEnter:
			g.confirmMigrationPreview()
			g.advanceOpenRow()
		}
	case ui.RowResearch:
		switch key {
		case ebiten.KeyArrowUp:
			g.researchCursor = (g.researchCursor + gameapi.TechCount - 1) % gameapi.TechCount
		case ebiten.KeyArrowDown:
			g.researchCursor = (g.researchCursor + 1) % gameapi.TechCount
		case ebiten.KeyEnter:
			g.chooseResearchTechnology(g.researchCursor)
			g.advanceOpenRow()
		}
	case ui.RowWorkforce:
		switch key {
		case ebiten.KeyArrowUp:
			g.assignmentRole = (g.assignmentRole + gameapi.AssignmentCount - 1) % gameapi.AssignmentCount
		case ebiten.KeyArrowDown:
			g.assignmentRole = (g.assignmentRole + 1) % gameapi.AssignmentCount
		case ebiten.KeyArrowLeft:
			g.adjustSelectedRole(-100, shift)
		case ebiten.KeyArrowRight:
			g.adjustSelectedRole(100, shift)
		case ebiten.KeyEnter:
			g.applyAssignmentDraft()
		}
	}
}

func (g *Game) adjustSelectedRole(deltaBP int, shift bool) {
	if shift {
		deltaBP *= 5
	}
	g.editAssignmentDraft(deltaBP)
}

// changeOpenRow moves the accordion by delta rows, wrapping, and records that
// the player chose the row so auto-advance yields.
func (g *Game) changeOpenRow(delta int) {
	count := int(ui.ChecklistRowCount)
	g.openRow = ui.ChecklistRow((int(g.openRow) + delta + count) % count)
	g.rowChosen = true
}

// escape peels one transient layer (spec §8) and reports whether it did; the
// caller opens the menu when nothing was peeled.
func (g *Game) escape() bool {
	switch {
	case g.hasMigrationPreview:
		g.clearMigrationPreview()
		g.showNotice("Migration choice cleared")
	case g.bandListOpen:
		g.bandListOpen = false
	case g.shortcutsOpen:
		g.shortcutsOpen = false
	default:
		return false
	}
	return true
}

func (g *Game) toggleNotesExpanded() {
	switch g.notesMode {
	case hud.NotesExpanded:
		g.setNotesMode(hud.NotesCompact)
	default:
		g.setNotesMode(hud.NotesExpanded)
	}
}
```

Add fields `shortcutsOpen bool` and `researchCursor gameapi.Tech`, and a stub `func (g *Game) toggleCameraOverride() {}` (Task 15 fills it). Add `ShortcutsOpen bool` to `hud.State` and set it from `g.shortcutsOpen` in `hudState`; Task 13 renders it as a modal window listing the keys in spec §8, and until then the field is inert. **Deviation from spec §8:** ebitenui v0.7.3's `TextArea` exposes `SetText`/`AppendText` but no scroll setter, so Shift+PgUp/PgDn drawer scrolling is not implemented; the drawer scrolls with the wheel. Record this in the DESIGN.md keyboard reference (Task 17).

In `Game.Update`, replace the whole gameplay keyboard section (from the Tab handler down to and including the Space handler) with a single `g.handleGameplayKeys()`, keeping the mouse click and hover code. In `handleSceneInput`'s `SceneGameplay` case, replace the two Escape branches with:

```go
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.escape() {
				g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneMenu)})
			}
			return true
		}
```

Remove the `W` handler, the `[`/`]` handler, and the gameplay `-`/`=` volume handler (Settings keeps its own). Delete `TestFieldNotesAndSplitHotkeysRemainDistinct` if it only asserted the removed legend text; keep it if it tests `handleGameplayHotkey`, which stays for `F`/`N`.

- [ ] **Step 4: Run, smoke, commit**

Run: `go test ./pkg/app/ -v && go build ./...` — Expected: PASS. Then run the real game briefly (`go run .`) and check: arrows move the cursor with Move open, PgDn opens Research, Up/Down then Enter picks a tech, PgDn again opens Workforce, Left/Right change the marked role, Space ends the turn, Esc opens the menu when nothing is pending.

```bash
git add pkg/app
git commit -m "Route arrows and Enter to the open checklist row; remove W, [, ] and gameplay volume keys

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 13: Scene overlays and the end scene on ebitenui

**Files:**
- Create: `pkg/hud/overlays.go`, `pkg/hud/end_scene.go`, `pkg/hud/overlays_test.go`
- Modify: `pkg/hud/panel.go` (remove `buildOverlay`/`buildEndScene` stubs), `pkg/app/hud_state.go` (`overlayState`), `pkg/app/hud_intents.go` (`handleOverlayIntent`), `pkg/app/game.go` (remove pointer handling inside `handleSceneInput`; end-scene click), `pkg/app/game_test.go` (un-skip and rewrite the four overlay tests)

**Interfaces:**
- Consumes: `State.Overlay`, `State.Ending`, `State.ShortcutsOpen`.
- Produces: `handles.overlayButtons []*widget.Button` in row order for tests; `handles.newCampaign *widget.Button`; `func (g *Game) overlayState() hud.OverlayState`; `func (g *Game) handleOverlayIntent(intent hud.Intent)`.

- [ ] **Step 1: Write the failing tests**

`pkg/hud/overlays_test.go`:

```go
package hud

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestMenuOverlayButtonsEmitNavigationIntents(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneMenu}
	panel.Update(state)
	if panel.handles.overlay == nil || len(panel.handles.overlayButtons) != 6 {
		t.Fatalf("menu buttons = %d", len(panel.handles.overlayButtons))
	}
	want := []IntentKind{IntentBack, IntentOpenStorage, IntentOpenStorage, IntentOpenSettings, IntentSetNotesMode, IntentReturnToTitle}
	for index, kind := range want {
		panel.handles.overlayButtons[index].Click()
		intents := panel.Update(state)
		if len(intents) != 1 || intents[0].Kind != kind {
			t.Fatalf("menu button %d = %+v, want %v", index, intents, kind)
		}
		if index == 1 && !intents[0].Save || index == 2 && intents[0].Save {
			t.Fatalf("storage mode for button %d = %+v", index, intents)
		}
	}
}

func TestStorageOverlayRowsCarrySlots(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneStorage, StorageHeading: "Load / Delete"}
	state.Overlay.StorageRows[0] = StorageRow{Label: "Manual 1", Detail: "Turn 4 · 78,800 BP · sapiens 490", Slot: 1, Occupied: true, Writable: true}
	state.Overlay.StorageRows[1] = StorageRow{Label: "Quick", Detail: "Empty", Slot: 99}
	panel.Update(state)
	if len(panel.handles.overlayButtons) != 2 {
		t.Fatalf("rows with labels = %d, want 2", len(panel.handles.overlayButtons))
	}
	panel.handles.overlayButtons[0].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentLoadSlot || intents[0].Slot != 1 {
		t.Fatalf("occupied row click = %+v", intents)
	}
	if !panel.handles.overlayButtons[1].GetWidget().Disabled {
		t.Fatal("empty slot is clickable in the load browser")
	}
	if len(panel.handles.deleteButtons) != 2 || !panel.handles.deleteButtons[1].GetWidget().Disabled {
		t.Fatal("delete controls: want one per labelled row, disabled for empty slots")
	}
}

func TestEndSceneShowsNewCampaignButton(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	frame.CampaignResult = gameapi.Extinction
	state := testState(frame, 1)
	state.Ending = ui.CampaignEndScene(frame)
	panel.Update(state)
	if panel.handles.newCampaign == nil {
		t.Fatal("end scene has no New Campaign button")
	}
	panel.handles.newCampaign.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentNewCampaign {
		t.Fatalf("new campaign = %+v", intents)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run 'TestMenuOverlay|TestStorageOverlay|TestEndScene'` — Expected: FAIL.

- [ ] **Step 3: Implement `overlays.go`**

```go
package hud

import (
	"fmt"
	"image"

	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

const (
	overlayX      = 340.0
	overlayY      = 150.0
	overlayWidth  = 600.0
	overlayHeight = 420.0
)

// buildOverlay shows the modal for the current non-gameplay scene, or the
// shortcut sheet, as an ebitenui window that blocks input beneath it.
func (p *Panel) buildOverlay(state State) {
	t := p.theme
	var content *widget.Container
	switch {
	case state.Overlay.Scene == ui.SceneTitle:
		content = p.menuList("Africa 2 Ice: Paleolithic Dispersal", "Guide Homo sapiens from East Africa, 80,000–20,000 BP.", []menuEntry{
			{"Continue · Enter", Intent{Kind: IntentContinue}},
			{"New Campaign · N", Intent{Kind: IntentNewCampaign}},
			{"Load a checkpoint · L", Intent{Kind: IntentOpenStorage}},
		})
	case state.Overlay.Scene == ui.SceneMenu:
		notes := "Show Field Notes · F"
		mode := NotesCompact
		if state.NotesMode != NotesHidden {
			notes, mode = "Hide Field Notes · F", NotesHidden
		}
		content = p.menuList("Game Menu", "Turns advance only when you explicitly end them.", []menuEntry{
			{"Back to game · Esc", Intent{Kind: IntentBack}},
			{"Save slots · S", Intent{Kind: IntentOpenStorage, Save: true}},
			{"Load or delete slots · L", Intent{Kind: IntentOpenStorage}},
			{"Settings · O", Intent{Kind: IntentOpenSettings}},
			{notes, Intent{Kind: IntentSetNotesMode, Notes: mode}},
			{"Return to title · T", Intent{Kind: IntentReturnToTitle}},
		})
	case state.Overlay.Scene == ui.SceneStorage:
		content = p.storageList(state)
	case state.Overlay.Scene == ui.SceneSettings:
		content = p.settingsPanel(state)
	case state.ShortcutsOpen:
		content = p.shortcutSheet()
	default:
		return
	}
	rect := image.Rectangle(p.rect(overlayX, overlayY, overlayWidth, overlayHeight))
	window := widget.NewWindow(widget.WindowOpts.Contents(content), widget.WindowOpts.Modal(), widget.WindowOpts.CloseMode(widget.NONE), widget.WindowOpts.Location(rect))
	p.ui.AddWindow(window)
	p.handles.overlay = window
}

type menuEntry struct {
	label  string
	intent Intent
}

func (p *Panel) overlayFrame(heading, help string) *widget.Container {
	t := p.theme
	frame := t.column(10, t.insets(24, 28, 28, 20), bordered(colorPanel, colorGoldDeep, t.px(2)), widget.WidgetOpts.MinSize(t.px(overlayWidth), t.px(overlayHeight)))
	frame.AddChild(t.label(heading, 25, colorTitle))
	if help != "" {
		frame.AddChild(t.label(help, 11, colorDim))
	}
	return frame
}

func (p *Panel) menuList(heading, help string, entries []menuEntry) *widget.Container {
	t := p.theme
	frame := p.overlayFrame(heading, help)
	for _, entry := range entries {
		intent := entry.intent
		button := t.button(entry.label, 14, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		frame.AddChild(button)
	}
	return frame
}

func (p *Panel) storageList(state State) *widget.Container {
	t := p.theme
	help := "Click a slot · Delete removes it · Esc back"
	if state.Overlay.StorageBusy != "" {
		help = state.Overlay.StorageBusy
	}
	frame := p.overlayFrame(state.Overlay.StorageHeading, help)
	saving := state.Overlay.StorageHeading == "Save / Delete"
	for _, row := range state.Overlay.StorageRows {
		if row.Label == "" {
			continue
		}
		slot := row.Slot
		line := t.rowOf(8, stretch())
		var intent Intent
		switch {
		case saving:
			intent = Intent{Kind: IntentSaveSlot, Slot: slot}
		default:
			intent = Intent{Kind: IntentLoadSlot, Slot: slot}
		}
		button := t.button(fmt.Sprintf("%-10s %s", row.Label, row.Detail), 12, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().Disabled = state.Overlay.StorageBusy != "" || (saving && !row.Writable) || (!saving && !row.Occupied)
		button.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		remove := t.button("Delete", 10, colorRed, colorRed, func() { p.emit(Intent{Kind: IntentDeleteSlot, Slot: slot}) })
		remove.GetWidget().Disabled = state.Overlay.StorageBusy != "" || !row.Occupied
		p.handles.deleteButtons = append(p.handles.deleteButtons, remove)
		line.AddChild(button, remove)
		frame.AddChild(line)
	}
	frame.AddChild(t.button("Back · Esc", 12, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentBack}) }))
	return frame
}

func (p *Panel) settingsPanel(state State) *widget.Container {
	t := p.theme
	help := "Drag the slider · M mutes · F toggles notes · O or Esc back"
	if state.Overlay.SettingsDisabled {
		help = "Loading preferences… controls disabled · O/Esc back"
	}
	frame := p.overlayFrame("Settings", help)
	volume := t.rowOf(10, stretch())
	volume.AddChild(t.label(fmt.Sprintf("Master volume %3.0f%%", state.Overlay.MasterVolume*100), 13, colorText))
	slider := widget.NewSlider(
		widget.SliderOpts.Direction(widget.DirectionHorizontal),
		widget.SliderOpts.MinMax(0, 100),
		widget.SliderOpts.InitialCurrent(int(state.Overlay.MasterVolume*100+0.5)),
		widget.SliderOpts.Images(&widget.SliderTrackImage{Idle: solid(colorPanelEdge), Hover: solid(colorPanelEdge)}, t.buttonImages(colorGoldDeep)),
		widget.SliderOpts.FixedHandleSize(t.px(12)),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			p.emit(Intent{Kind: IntentSetVolume, Volume: float64(args.Current) / 100})
		}),
		widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(220), t.px(14))),
	)
	slider.GetWidget().Disabled = state.Overlay.SettingsDisabled
	volume.AddChild(slider)
	frame.AddChild(volume)
	mute := "Muted: off · M"
	if state.Overlay.Muted {
		mute = "Muted: on · M"
	}
	notes := "Field Notes: hidden · F"
	if state.NotesMode != NotesHidden {
		notes = "Field Notes: visible · F"
	}
	for _, entry := range []menuEntry{
		{mute, Intent{Kind: IntentToggleMute}},
		{notes, Intent{Kind: IntentSetNotesMode, Notes: NotesCompact}},
		{"Show first-turn guide", Intent{Kind: IntentShowGuide}},
		{"Back · Esc", Intent{Kind: IntentBack}},
	} {
		intent := entry.intent
		if intent.Kind == IntentSetNotesMode && state.NotesMode != NotesHidden {
			intent.Notes = NotesHidden
		}
		button := t.button(entry.label, 13, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().Disabled = state.Overlay.SettingsDisabled && intent.Kind != IntentBack
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		frame.AddChild(button)
	}
	return frame
}

func (p *Panel) shortcutSheet() *widget.Container {
	t := p.theme
	frame := p.overlayFrame("Keyboard shortcuts", "? or Esc closes")
	lines := []string{
		"Space  end turn        Tab / Shift+Tab  next / previous band",
		"PgUp / PgDn or Shift+Up/Down  change the open row",
		"Move row:  arrows steer the cursor · Enter queues · Esc clears",
		"Research row:  Up/Down highlight · Enter chooses · 1–9 direct",
		"Workforce row:  Up/Down pick a role · Left/Right or −/+ step 1% · Shift 5% · Enter/A apply · D discard",
		"N split · I interbreed · J cycle partner · G cycle trait note",
		"F notes · Shift+F expand notes · wheel scrolls notes · Z camera",
		"Ctrl/Cmd+S quick-save · F1–F3 save · Shift+F1–F3 load · M mute · Esc menu",
	}
	for _, line := range lines {
		frame.AddChild(t.label(line, 11.5, colorText))
	}
	frame.AddChild(t.button("Close · Esc", 12, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleShortcuts}) }))
	return frame
}
```

Add `IntentToggleShortcuts` to the intent kinds, `overlayButtons, deleteButtons []*widget.Button` to `handles`, and reset them in `rebuild`.

- [ ] **Step 4: Implement `end_scene.go`**

```go
package hud

import "github.com/ebitenui/ebitenui/widget"

// buildEndScene adds the New Campaign button over the render-drawn terminal
// scene; the text itself stays in pkg/render.
func (p *Panel) buildEndScene(state State) widget.PreferredSizeLocateableWidget {
	if !state.Ending.Visible {
		return nil
	}
	t := p.theme
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(490, 494, 300, 46))))
	button := t.button("New Campaign · N", 15, colorGoldDeep, colorBlack, func() { p.emit(Intent{Kind: IntentNewCampaign}) })
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.newCampaign = button
	holder.AddChild(button)
	return holder
}
```

Add `newCampaign *widget.Button` to `handles`. Remove the stubs from `panel.go`.

- [ ] **Step 5: App side — `overlayState` and `handleOverlayIntent`**

Replace the stub in `hud_state.go`:

```go
func (g *Game) overlayState() hud.OverlayState {
	overlay := hud.OverlayState{Scene: g.scenes.Current(), MasterVolume: g.settings.MasterVolume, Muted: g.settings.Muted, SettingsDisabled: g.settingsLoading}
	if overlay.Scene != ui.SceneStorage {
		return overlay
	}
	overlay.StorageHeading = "Load / Delete"
	if g.storageMode == storageBrowserSave {
		overlay.StorageHeading = "Save / Delete"
	}
	for index, slot := range storageBrowserSlots {
		row := hud.StorageRow{Label: storageSlotLabel(slot), Slot: slot, Detail: "Empty", Writable: slot >= 1 && slot <= 3}
		if metadata, exists := g.storageMetadata(slot); exists {
			row.Occupied = true
			row.Detail = fmt.Sprintf("Turn %d · %d BP · sapiens %d", metadata.Turn, metadata.YearBP, metadata.SapiensPopulation)
		}
		overlay.StorageRows[index] = row
	}
	switch {
	case g.storageListID != 0:
		overlay.StorageBusy = "Reading save metadata…"
	case g.storageOperationID != 0:
		overlay.StorageBusy = "Storage operation pending…"
	}
	return overlay
}
```

Replace the stub in `hud_intents.go`:

```go
func (g *Game) handleOverlayIntent(intent hud.Intent) {
	switch intent.Kind {
	case hud.IntentBack:
		g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
	case hud.IntentContinue:
		g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
	case hud.IntentNewCampaign:
		g.startNewCampaign()
		if g.scenes.Current() == ui.SceneTitle {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
		}
	case hud.IntentOpenStorage:
		mode := storageBrowserLoad
		if intent.Save {
			mode = storageBrowserSave
		}
		g.openStorageBrowser(mode)
	case hud.IntentOpenSettings:
		g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneSettings)})
	case hud.IntentReturnToTitle:
		if g.assignmentDraftDirty() {
			g.showNotice("Apply or discard workforce changes before returning to title")
			return
		}
		g.scenes.Reset()
		g.scenes.Push(ui.SceneTitle)
	case hud.IntentSaveSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.activateStorageSelection()
	case hud.IntentLoadSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.activateStorageSelection()
	case hud.IntentDeleteSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.deleteStorageSelection()
	case hud.IntentSetVolume:
		if !g.settingsLoading && intent.Volume != g.settings.MasterVolume {
			settings := g.settings
			settings.MasterVolume = intent.Volume
			g.updateUISettings(settings)
		}
	case hud.IntentToggleMute:
		g.toggleMute()
	case hud.IntentShowGuide:
		g.guide = ui.NewGuideState(false)
		if !g.settingsLoading {
			settings := g.settings
			settings.GuideDismissed = false
			g.updateUISettings(settings)
		}
	case hud.IntentToggleShortcuts:
		g.shortcutsOpen = !g.shortcutsOpen
	}
}

func storageIndexForSlot(slot int) int {
	for index, candidate := range storageBrowserSlots {
		if candidate == slot {
			return index
		}
	}
	return 0
}
```

In `handleSceneInput`, delete every `IsMouseButtonJustPressed`/`IsMouseButtonPressed` block (title rows, settings slider, mute/notes rows); keyboard handling stays. Delete `menuOverlayForRender`. In `Game.Update`'s terminal-campaign branch, remove `mouseStartsCampaign` and keep only the `N` key; the button arrives as `IntentNewCampaign`. Delete `NewCampaignButtonContains` usage. Set `ShortcutsOpen: g.shortcutsOpen` in `hudState`.

Rewrite the four skipped tests against `overlayState()`:

```go
func TestTitleAndGameMenuExposeCampaignNavigation(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.scenes.Push(ui.SceneTitle)
	if game.overlayState().Scene != ui.SceneTitle {
		t.Fatal("title scene not reported")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentContinue}})
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("Continue did not pop the title")
	}
	game.scenes.Push(ui.SceneMenu)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentOpenSettings}})
	if game.scenes.Current() != ui.SceneSettings {
		t.Fatal("Settings intent did not open settings")
	}
}
```

For the storage tests, replace `overlay := game.menuOverlayForRender()` with `overlay := game.overlayState()` and assert on `overlay.StorageRows[0].Detail` containing `"Turn 4"`, `overlay.StorageRows[5].Detail` containing `"Turn 8"`, `overlay.StorageRows[3].Detail == "Empty"`; replace `game.storageSelection = 5; game.activateStorageSelection()` with `game.handleIntents([]hud.Intent{{Kind: hud.IntentLoadSlot, Slot: 102}})`. For settings, assert `overlayState().MasterVolume == 0.7` and `Muted`. `TestGameMenuDescribesTurnBasedBehavior` becomes a `pkg/hud` test asserting the menu help text contains "explicitly end" (add it to `overlays_test.go` by inspecting the second child label of `panel.handles.overlay.GetContainer()` or simply by keeping the string in a package constant `menuHelp` and asserting on it).

- [ ] **Step 6: Build, test, run, commit**

Run: `go build ./... && go vet ./... && go test ./... && golangci-lint run` — Expected: PASS, no skips left. Run `go run .` and click through title → game → Esc menu → Settings slider → Back; open Load browser and confirm rows; end a campaign in a test save if available, or trust `TestEndSceneShowsNewCampaignButton`.

```bash
git add -A pkg/hud pkg/app
git commit -m "Scene overlays and the end-scene button as ebitenui windows

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

Phase 2 ends here: the web smoke test must still pass — `./build_web.sh --dev && npm --prefix tools/web-e2e test` (requires the Playwright browser installed once with `npm --prefix tools/web-e2e run install-browser`).

---

## Phase 3 — Two-state camera

### Task 14: `render.Camera` and camera-aware map geometry

**Files:**
- Create: `pkg/render/camera.go`, `pkg/render/camera_test.go`
- Modify: `pkg/render/map.go` (`tilePoint`, `PickTile`, `MapTileAt`, `drawTerrain`, `drawReachableTiles`, `drawMigrationPreview`, `drawQueuedMigrations`, `drawEscarpments`/`escarpmentLine`, band markers, `mapFrameKey`, map-area clipping), `pkg/render/map_test.go`

**Interfaces:**
- Produces:
  - `type CameraMode uint8` — `CameraOverview`, `CameraFocus`.
  - `type Camera struct { Mode CameraMode; CenterTile gameapi.TileID; Progress float64 }` — `Progress` is the blend toward Focus (0 overview, 1 focus).
  - `const FocusScale = 3.0`, `const CameraTransitionTicks = 15`.
  - `func (c Camera) Step() Camera` — moves `Progress` one tick toward `Mode`.
  - `func (c Camera) Settled() bool`
  - `type MapGeometry struct { OriginX, OriginY, Cell float32 }`
  - `func CameraGeometry(camera Camera, frame *gameapi.Frame, visibleHeight float64) MapGeometry` — `visibleHeight` is the map-area height in DIPs above the drawer (626 minus drawer height).
  - `func (g MapGeometry) TilePoint(tile gameapi.Tile) (float32, float32)` and `func (g MapGeometry) TileAt(x, y int) (gameapi.TileID, bool)`.
  - `func MapTileAt(camera Camera, frame *gameapi.Frame, visibleHeight float64, x, y int) (gameapi.TileID, bool)` replaces the old two-argument function.
  - `func (scene *MapScene) SetCamera(camera Camera, visibleHeight float64)`; `func (scene *MapScene) SetGuideHighlight(on bool)` (dashed box around the reachable cluster, Task 16 turns it on).

- [ ] **Step 1: Write the failing tests**

Create `pkg/render/camera_test.go`:

```go
package render

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func cameraFrame() *gameapi.Frame {
	frame := &gameapi.Frame{Tiles: make([]gameapi.Tile, TerrainGridWidth*TerrainGridHeight)}
	for y := 0; y < TerrainGridHeight; y++ {
		for x := 0; x < TerrainGridWidth; x++ {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			frame.Tiles[id] = gameapi.Tile{ID: id, X: x, Y: y, Land: true, Explored: true}
		}
	}
	return frame
}

func TestOverviewGeometryMatchesTheLockedGrid(t *testing.T) {
	geometry := CameraGeometry(Camera{}, cameraFrame(), 626)
	if geometry.OriginX != mapOriginX || geometry.OriginY != mapOriginY || geometry.Cell != mapTileSize {
		t.Fatalf("overview geometry = %+v", geometry)
	}
	x, y := geometry.TilePoint(gameapi.Tile{X: 3, Y: 2})
	if x != mapOriginX+3*mapTileSize+mapTileSize/2 || y != mapOriginY+2*mapTileSize+mapTileSize/2 {
		t.Fatalf("overview tile point = (%.1f, %.1f)", x, y)
	}
}

func TestFocusGeometryCentersTheTileAndClampsToTheGrid(t *testing.T) {
	frame := cameraFrame()
	center := gameapi.TileID(30*TerrainGridWidth + 40)
	geometry := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: center, Progress: 1}, frame, 626)
	if geometry.Cell != mapTileSize*FocusScale {
		t.Fatalf("focus cell = %.1f", geometry.Cell)
	}
	x, y := geometry.TilePoint(frame.Tiles[center])
	if math.Abs(float64(x)-(mapOriginX+864/2)) > 0.6 || math.Abs(float64(y)-(mapOriginY+626/2)) > 0.6 {
		t.Fatalf("focused tile center = (%.1f, %.1f), want the middle of the visible map area", x, y)
	}
	corner := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: 0, Progress: 1}, frame, 626)
	if corner.OriginX != mapOriginX || corner.OriginY != mapOriginY {
		t.Fatalf("top-left focus should clamp to the map origin, got %+v", corner)
	}
	last := gameapi.TileID(len(frame.Tiles) - 1)
	farCorner := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: last, Progress: 1}, frame, 626)
	if right := farCorner.OriginX + float32(TerrainGridWidth)*farCorner.Cell; math.Abs(float64(right)-(mapOriginX+864)) > 0.6 {
		t.Fatalf("bottom-right focus should clamp so the grid's right edge meets the map edge, got right %.1f", right)
	}
	if bottom := farCorner.OriginY + float32(TerrainGridHeight)*farCorner.Cell; math.Abs(float64(bottom)-(mapOriginY+626)) > 0.6 {
		t.Fatalf("bottom-right focus should clamp to the visible height, got bottom %.1f", bottom)
	}
}

func TestTileAtRoundTripsTilePointInBothModesAndMidTransition(t *testing.T) {
	frame := cameraFrame()
	for _, camera := range []Camera{
		{},
		{Mode: CameraFocus, CenterTile: 2000, Progress: 1},
		{Mode: CameraFocus, CenterTile: 2000, Progress: 0.4},
	} {
		geometry := CameraGeometry(camera, frame, 500)
		for _, id := range []gameapi.TileID{2000, 2001, 2000 + TerrainGridWidth} {
			x, y := geometry.TilePoint(frame.Tiles[id])
			if got, ok := geometry.TileAt(int(x), int(y)); !ok || got != id {
				t.Fatalf("camera %+v: TileAt(TilePoint(%d)) = (%d, %t)", camera, id, got, ok)
			}
		}
		if _, ok := geometry.TileAt(mapOriginX-1, mapOriginY); ok {
			t.Fatalf("camera %+v: point left of the map picked a tile", camera)
		}
		if _, ok := geometry.TileAt(mapOriginX+10, mapOriginY+501); ok {
			t.Fatalf("camera %+v: point below the visible area picked a tile", camera)
		}
	}
}

func TestCameraStepsTowardItsModeAndSettles(t *testing.T) {
	camera := Camera{Mode: CameraFocus}
	for tick := 0; tick < CameraTransitionTicks; tick++ {
		if camera.Settled() {
			t.Fatalf("settled after %d ticks", tick)
		}
		camera = camera.Step()
	}
	if !camera.Settled() || camera.Progress != 1 {
		t.Fatalf("after %d ticks = %+v", CameraTransitionTicks, camera)
	}
	camera.Mode = CameraOverview
	camera = camera.Step()
	if camera.Progress >= 1 || camera.Settled() {
		t.Fatalf("switching back did not start moving: %+v", camera)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/render/ -run 'TestOverviewGeometry|TestFocusGeometry|TestTileAtRoundTrips|TestCameraSteps'` — Expected: FAIL.

- [ ] **Step 3: Implement `camera.go`**

```go
package render

import "github.com/adsouza/africa2ice/pkg/gameapi"

// CameraMode is one of the two map views (spec §6). There is no free pan.
type CameraMode uint8

const (
	CameraOverview CameraMode = iota
	CameraFocus
)

const (
	FocusScale            = 3.0 // 24 px cells
	CameraTransitionTicks = 15  // 250 ms at 60 TPS
	mapAreaWidth          = 864.0 // map rectangle x 20–884
	mapAreaHeight         = 626.0 // map rectangle y 74–700
)

// Camera is UI-local presentation state owned by the application. Progress is
// the blend toward Focus so a mode change animates from wherever it was.
type Camera struct {
	Mode       CameraMode
	CenterTile gameapi.TileID
	Progress   float64
}

func (camera Camera) target() float64 {
	if camera.Mode == CameraFocus {
		return 1
	}
	return 0
}

// Step advances the blend one tick toward the mode.
func (camera Camera) Step() Camera {
	step := 1.0 / CameraTransitionTicks
	target := camera.target()
	switch {
	case camera.Progress < target:
		camera.Progress = min(target, camera.Progress+step)
	case camera.Progress > target:
		camera.Progress = max(target, camera.Progress-step)
	}
	return camera
}

func (camera Camera) Settled() bool { return camera.Progress == camera.target() }

// MapGeometry is the resolved cell size and grid origin for one frame.
type MapGeometry struct {
	OriginX       float32
	OriginY       float32
	Cell          float32
	visibleHeight float32 // TileAt's bottom bound; 0 means the full map area
}

func (geometry MapGeometry) visibleBottom() float32 {
	if geometry.visibleHeight <= 0 {
		return mapOriginY + float32(mapAreaHeight)
	}
	return mapOriginY + geometry.visibleHeight
}

// CameraGeometry blends the overview grid with a 3× view centred on the
// camera's tile, clamped so the grid never shows space beyond its edges inside
// the visible map area (x 20–884, y 74 to 74+visibleHeight).
func CameraGeometry(camera Camera, frame *gameapi.Frame, visibleHeight float64) MapGeometry {
	overview := MapGeometry{OriginX: mapOriginX, OriginY: mapOriginY, Cell: mapTileSize, visibleHeight: float32(visibleHeight)}
	if camera.Progress <= 0 || frame == nil || int(camera.CenterTile) >= len(frame.Tiles) {
		return overview
	}
	cell := mapTileSize * FocusScale
	tile := frame.Tiles[camera.CenterTile]
	centerX := (float64(tile.X) + 0.5) * cell
	centerY := (float64(tile.Y) + 0.5) * cell
	originX := mapOriginX + mapAreaWidth/2 - centerX
	originY := mapOriginY + visibleHeight/2 - centerY
	originX = max(min(originX, mapOriginX), mapOriginX+mapAreaWidth-float64(TerrainGridWidth)*cell)
	originY = max(min(originY, mapOriginY), mapOriginY+visibleHeight-float64(TerrainGridHeight)*cell)
	blend := camera.Progress
	return MapGeometry{
		OriginX: float32(lerp(float64(overview.OriginX), originX, blend)),
		OriginY: float32(lerp(float64(overview.OriginY), originY, blend)),
		Cell:    float32(lerp(float64(overview.Cell), cell, blend)),

		visibleHeight: float32(visibleHeight),
	}
}

// TilePoint is the centre of a tile's cell under this geometry.
func (geometry MapGeometry) TilePoint(tile gameapi.Tile) (float32, float32) {
	return geometry.OriginX + (float32(tile.X)+0.5)*geometry.Cell, geometry.OriginY + (float32(tile.Y)+0.5)*geometry.Cell
}

// TileAt inverts TilePoint for a logical point inside the map area.
func (geometry MapGeometry) TileAt(x, y int) (gameapi.TileID, bool) {
	fx, fy := float32(x), float32(y)
	if fx < mapOriginX || fy < mapOriginY || fx >= mapOriginX+float32(mapAreaWidth) || fy >= geometry.visibleBottom() {
		return 0, false
	}
	gridX := int((fx - geometry.OriginX) / geometry.Cell)
	gridY := int((fy - geometry.OriginY) / geometry.Cell)
	if fx < geometry.OriginX || fy < geometry.OriginY || gridX < 0 || gridX >= TerrainGridWidth || gridY < 0 || gridY >= TerrainGridHeight {
		return 0, false
	}
	return gameapi.TileID(gridY*TerrainGridWidth + gridX), true
}

// MapTileAt is the camera-aware pick used by hover and clicks.
func MapTileAt(camera Camera, frame *gameapi.Frame, visibleHeight float64, x, y int) (gameapi.TileID, bool) {
	return CameraGeometry(camera, frame, visibleHeight).TileAt(x, y)
}
```

`mapPixelWidth`/`mapPixelHeight` (768/512) remain the terrain layer's overview size; the map *area* is 864 × 626 because the panel starts at x = 908 and the bottom strip is gone.

- [ ] **Step 4: Thread the geometry through `map.go`**

- Add `camera Camera`, `visibleHeight float64`, `guideHighlight bool` to `MapScene`; `SetCamera(camera, visibleHeight)` and `SetGuideHighlight(on)`; add all three to `mapFrameKey`.
- Add `func (scene *MapScene) geometry(frame *gameapi.Frame) MapGeometry { return CameraGeometry(scene.camera, frame, scene.visibleHeight) }` and compute it once at the top of `drawFrame`, passing it to the overlay draws.
- `tilePoint(tile)` becomes `geometry.TilePoint(tile)`; `PickTile(x, y)` becomes `MapTileAt(scene.camera, scene.lastFrame, scene.visibleHeight, x, y)` where `lastFrame` is stored in `Draw`.
- Every use of `mapTileSize` in overlays (`drawReachableTiles`, `drawMigrationPreview`, hover tint, `escarpmentLine`, passage glyph and band marker radii) uses `geometry.Cell` instead; scale marker radii by `geometry.Cell/mapTileSize` so bands stay proportionate in focus.
- `drawTerrain`: keep the 1× terrain cache; draw it with `options.GeoM.Scale(float64(geometry.Cell)/mapTileSize, …)` then translate to `(geometry.OriginX*scale, geometry.OriginY*scale)`, and set `options.Filter = ebiten.FilterNearest`.
- Clip every map layer to the visible map area: in `drawFrame`, build `mapCanvas := logicalCanvas{image: screen.image.SubImage(image.Rect(round(20*s), round(74*s), round(884*s), round((74+visibleHeight)*s))).(*ebiten.Image), scale: screen.scale}` and draw terrain, overlays, markers, arrows, and the hover tint into `mapCanvas` (SubImage shares the parent's coordinate space, so no coordinate changes are needed). Timeline, legend, notice, end scene keep drawing to `screen`.
- Guide highlight: when `scene.guideHighlight` and the selected band has candidates, stroke a dashed gold rectangle (alternate 6 px on / 4 px off using `vector.StrokeLine` segments) around the bounding box of the candidate tiles, inflated by half a cell.

Update `map_test.go`: `TestMapTileAt` calls `MapTileAt(Camera{}, frame, 626, x, y)`; `TestTopDownTerrainKeepsColorPickingAndMarkersOnTheSameGrid` uses `scene.SetCamera(Camera{}, 626)` before picking; add a subtest with `Camera{Mode: CameraFocus, CenterTile: …, Progress: 1}` asserting `PickTile(TilePoint(tile)) == tile`.

- [ ] **Step 5: Run, commit**

Run: `go test ./pkg/render/ -v && go build ./...` — Expected: PASS (app still compiles because `SetCamera` is optional; `exploredHoverTile` in `pkg/app` calls `render.MapTileAt` — update it to `render.MapTileAt(g.camera, frame, g.mapVisibleHeight(), x, y)` in Task 15; for now pass `render.Camera{}` and `626`).

```bash
git add pkg/render pkg/app/game.go
git commit -m "render: two-state camera geometry with clamped focus and clipped map layers

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 15: Camera behaviour in the app and the Overview/Focus control

**Files:**
- Modify: `pkg/app/game.go` (fields `camera render.Camera`, `cameraOverride bool`; `Update` camera step; `Draw` passes camera; `exploredHoverTile`; `resetDisclosure` clears override), `pkg/app/hud_state.go` (`Camera` state, `mapVisibleHeight`), `pkg/app/input.go` (`toggleCameraOverride` real), `pkg/app/hud_intents.go` (`IntentCameraToggle`)
- Modify: `pkg/hud/panel.go` (`buildCameraButton` placed at the map's top-right corner), `pkg/hud/panel_test.go`
- Test: `pkg/app/camera_test.go`

**Interfaces:**
- Produces: `func (g *Game) desiredCameraMode() render.CameraMode`; `func (g *Game) stepCamera()`; `func (g *Game) mapVisibleHeight() float64` = 626 − drawer height for `notesMode` (0 / 102 / 300); `handles.camera *widget.Button`.

- [ ] **Step 1: Write the failing tests**

`pkg/app/camera_test.go`:

```go
package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestCameraFocusesWhileTheMoveRowIsOpenAndTheActionIsAvailable(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("fresh band with Move open should request Focus")
	}
	game.openRow, game.rowChosen = ui.RowResearch, true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Research open should return to Overview")
	}
	game.openRow = ui.RowMove
	game.frame.Bands[0].SpatialActionUsed = true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("spent action should return to Overview")
	}
	game.frame.Bands[0].SpatialActionUsed = false
	game.toggleCameraOverride()
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Z should invert the automatic choice")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 7}})
	game.resetDisclosure()
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("override should reset with disclosure")
	}
}

func TestCameraStepsEachUpdateAndTracksTheDrawer(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	for tick := 0; tick < render.CameraTransitionTicks; tick++ {
		game.stepCamera()
	}
	if !game.camera.Settled() || game.camera.Mode != render.CameraFocus || game.camera.CenterTile != game.frame.Bands[0].TileID {
		t.Fatalf("camera after transition = %+v", game.camera)
	}
	game.notesMode = hud.NotesExpanded
	if game.mapVisibleHeight() != 626-300 {
		t.Fatalf("visible height with expanded drawer = %.0f", game.mapVisibleHeight())
	}
	game.notesMode = hud.NotesHidden
	if game.mapVisibleHeight() != 626 {
		t.Fatalf("visible height with hidden drawer = %.0f", game.mapVisibleHeight())
	}
}
```

`pkg/hud/panel_test.go` addition:

```go
func TestCameraButtonReflectsFocusAndEmitsToggle(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Camera = CameraState{FocusAvailable: true, Focused: true}
	panel.Update(state)
	if panel.handles.camera == nil || panel.handles.camera.Text().Label != "Overview · Z" {
		t.Fatal("focused camera should offer Overview")
	}
	panel.handles.camera.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentCameraToggle {
		t.Fatalf("camera click = %+v", intents)
	}
	state.Camera = CameraState{}
	panel.Update(state)
	if panel.handles.camera != nil {
		t.Fatal("camera button shown when focus is unavailable")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/app/ -run TestCamera; go test ./pkg/hud/ -run TestCameraButton` — Expected: FAIL.

- [ ] **Step 3: Implement**

`pkg/app/game.go` additions:

```go
// desiredCameraMode applies spec §6: focus while the Move row is open and the
// selected sapiens band still has its spatial action, inverted by Z.
func (g *Game) desiredCameraMode() render.CameraMode {
	band := g.selected()
	auto := g.openRow == ui.RowMove && band != nil && band.Species == gameapi.HomoSapiens && !ui.MoveDone(*band)
	if g.cameraOverride {
		auto = !auto
	}
	if auto {
		return render.CameraFocus
	}
	return render.CameraOverview
}

// stepCamera runs once per Update: retarget, then advance the transition.
func (g *Game) stepCamera() {
	g.camera.Mode = g.desiredCameraMode()
	if band := g.selected(); band != nil {
		g.camera.CenterTile = band.TileID
	}
	g.camera = g.camera.Step()
}

func (g *Game) mapVisibleHeight() float64 {
	switch g.notesMode {
	case hud.NotesCompact:
		return 626 - 102
	case hud.NotesExpanded:
		return 626 - 300
	default:
		return 626
	}
}

func (g *Game) toggleCameraOverride() { g.cameraOverride = !g.cameraOverride }
```

Call `g.stepCamera()` in `Update` right before `g.handleIntents(g.panel.Update(g.hudState()))` (so the camera moves even while an overlay is open). In `resetDisclosure` add `g.cameraOverride = false`. In `Draw`, before `g.scene.Draw`, call `g.scene.SetCamera(g.camera, g.mapVisibleHeight())`. Change `exploredHoverTile(frame, x, y, inside)` to a method `g.exploredHoverTile(x, y, inside)` using `render.MapTileAt(g.camera, g.frame, g.mapVisibleHeight(), x, y)` and update `TestExploredHoverTileRejectsFogAndCoordinatesOutsideTheMap` to call the method. Remove the Task 12 stub `toggleCameraOverride`. In `hudState`, set:

```go
	if band := g.selected(); band != nil {
		state.Camera = hud.CameraState{FocusAvailable: band.Species == gameapi.HomoSapiens && !ui.MoveDone(*band), Focused: g.camera.Mode == render.CameraFocus}
	}
```

In `hud_intents.go` add `case hud.IntentCameraToggle: g.toggleCameraOverride()`.

`pkg/hud/panel.go`: in `rebuild`, after the drawer, add

```go
	if state.Camera.FocusAvailable {
		p.root.AddChild(p.buildCameraButton(state))
	}
```

and

```go
// buildCameraButton is the map-corner Overview/Focus control (spec §6).
func (p *Panel) buildCameraButton(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	label := "Focus · Z"
	if state.Camera.Focused {
		label = "Overview · Z"
	}
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapRight-118, 80, 110, 22))))
	button := t.button(label, 9.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentCameraToggle}) })
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.camera = button
	holder.AddChild(button)
	return holder
}
```

Add `camera *widget.Button` to `handles`.

- [ ] **Step 4: Run, look, commit**

Run: `go test ./... && go build ./...` — Expected: PASS. Run `go run .`: on a new campaign the map should zoom onto the first band over a quarter second, follow Tab between bands, zoom out when a move is queued or Research opens, and `Z` or the corner button should flip it. Hover and click tiles while zoomed and confirm the HERE/TARGET column and queued arrow land on the tile under the pointer.

```bash
git add pkg/app pkg/hud
git commit -m "Camera focuses on the selected band during destination choice; Z and a corner button override

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Phase 4 — Guide card, DESIGN.md, final verification

### Task 16: The first-turn guide card

**Files:**
- Create: `pkg/hud/guide_card.go`
- Modify: `pkg/hud/panel.go` (remove the `buildGuideCard` stub), `pkg/hud/panel_test.go`, `pkg/app/game.go` (`Draw` sets the map guide highlight; `startNewCampaign` resets the guide when not dismissed)

**Interfaces:**
- Consumes: `ui.GuideState` (`Visible`, `Progress`, `Title`, `Body`, `Step`), `IntentGuideNext`, `IntentGuideDismiss`.
- Produces: `handles.guideNext`, `handles.guideX`.

- [ ] **Step 1: Write the failing test**

```go
func TestGuideCardShowsStepProgressAndOnlyXDismisses(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Guide = ui.NewGuideState(false)
	panel.Update(state)
	if panel.handles.guideNext == nil || panel.handles.guideX == nil {
		t.Fatal("guide card controls missing on a fresh campaign")
	}
	panel.handles.guideNext.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentGuideNext {
		t.Fatalf("Next = %+v", intents)
	}
	panel.handles.guideX.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentGuideDismiss {
		t.Fatalf("× = %+v", intents)
	}
	state.Guide = ui.GuideState{Step: ui.GuideClosing}
	panel.Update(state)
	if panel.handles.guideNext != nil || panel.handles.guideX == nil {
		t.Fatal("closing card should offer only the ×")
	}
	state.Guide = ui.NewGuideState(true)
	panel.Update(state)
	if panel.handles.guideX != nil {
		t.Fatal("dismissed guide still rendered")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./pkg/hud/ -run TestGuideCard` — Expected: FAIL (`handles.guideNext` never set).

- [ ] **Step 3: Implement `guide_card.go`**

```go
package hud

import (
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

// buildGuideCard is the dismissible first-turn guide (spec §7). It sits in the
// column's slack above the checklist and never blocks input.
func (p *Panel) buildGuideCard(state State) widget.PreferredSizeLocateableWidget {
	if !state.Guide.Visible() {
		return nil
	}
	t := p.theme
	card := t.column(5, t.insets(8, 12, 12, 8), bordered(colorGuide, colorGold, t.px(1)), stretch())
	head := t.rowOf(6, stretch())
	head.AddChild(t.label(state.Guide.Title(), 9, colorGold))
	current, total := state.Guide.Progress()
	bar := t.rowOf(2)
	for step := 1; step <= total; step++ {
		fill := colorPanelEdge
		if step <= current {
			fill = colorGold
		}
		bar.AddChild(widget.NewContainer(widget.ContainerOpts.BackgroundImage(solid(fill)), widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(18), t.px(4)))))
	}
	head.AddChild(bar)
	x := t.button("×", 11, colorGold, colorGold, func() { p.emit(Intent{Kind: IntentGuideDismiss}) })
	x.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	p.handles.guideX = x
	head.AddChild(x)
	card.AddChild(head)
	card.AddChild(t.wrapped(state.Guide.Body(), 10, colorTitle, panelWidth-2*panelPadding-30))
	if state.Guide.Step < ui.GuideClosing {
		next := t.button("Next ▸", 10, colorGold, colorGold, func() { p.emit(Intent{Kind: IntentGuideNext}) })
		p.handles.guideNext = next
		card.AddChild(next)
	}
	return card
}
```

Add `guideNext, guideX *widget.Button` to `handles` (Task 7 declared them; confirm) and delete the stub in `panel.go`.

- [ ] **Step 4: Map highlight and new-campaign reset**

In `Game.Draw` before `g.scene.Draw`: `g.scene.SetGuideHighlight(g.guide.Step == ui.GuideMove && g.frame.CampaignResult == gameapi.Ongoing)`. In `startNewCampaign`, after `resetDisclosure()`: `if !g.settings.GuideDismissed { g.guide = ui.NewGuideState(false) }`. Add an app test:

```go
func TestGuideFollowsTheTurnAndPersistsDismissal(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	if game.guide.Step != ui.GuideMove {
		t.Fatalf("fresh guide = %+v", game.guide)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}})
	frame := migrationPreviewFrame()
	frame.Bands[0].HasQueuedMigration, frame.Bands[0].QueuedMigration = true, 2
	stub.frame = frame
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}}) // second apply returns the queued frame
	if game.guide.Step != ui.GuideResearch {
		t.Fatalf("guide after a queued move = %+v", game.guide)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentGuideDismiss}})
	if game.guide.Visible() || !game.settings.GuideDismissed {
		t.Fatalf("dismissal not persisted: guide %+v settings %+v", game.guide, game.settings)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentShowGuide}})
	if game.guide.Step != ui.GuideMove || game.settings.GuideDismissed {
		t.Fatal("Show first-turn guide did not restore the card")
	}
}
```

If the stub's first `Apply` already returns a frame with the queued flag, drop the duplicate intent line. Run: `go test ./pkg/app/ ./pkg/hud/` — Expected: PASS.

- [ ] **Step 5: Screenshot check and commit**

```bash
go run . -screenshot /tmp/hud-turn0.png -turns 0
```

The screenshot runner starts with default settings, so the guide card should be visible above THIS TURN with step 1 of 4 and a dashed highlight around the reachable cluster on the map.

```bash
git add pkg/hud pkg/app
git commit -m "First-turn guide card with step progress, Next, and a persisted ×

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 17: DESIGN.md rewrite for the new HUD contract

**Files:**
- Modify: `docs/DESIGN.md`

DESIGN.md must stand alone; the spec is the source text. Make each edit, then re-read the section once for internal consistency.

- [ ] **Step 1: §"Gameplay stats layout" (starts at the `### Gameplay stats layout` heading, currently ~line 6829)**

Replace the bullets "Persistent top bar", "Selected-band inspector header", "Band-selection keys and attention order", and "Band details below the header" with prose ported from spec §4 and §5: the panel column (header, chips, band line, details, guide, THIS TURN rows, End turn, footer); chip coloring by the Move predicate with `!`/`!!` text prefixes and the eight-chip overflow `+N` list; the three rows and their Done predicates (spec §5.1 table); the default-open rule (§5.2); End turn hard and soft blocks (§5.3); details contents; the HERE/TARGET comparison with TARGET precedence and the amber/red tiers (§5.4 table, as a Markdown table). Keep unchanged: warning tiers definition, attention order, Tab/Shift+Tab, terrain legend, escarpment overlay, reachability overlay, queued-migration marker, keyboard migration, keyboard splitting, rejected-destination feedback. In "Keyboard migration" change "The persistent controls legend names the arrow, `Enter`, and `Esc` bindings" to "The Move row's hint line and the `?` shortcut sheet name the arrow, `Enter`, and `Esc` bindings; arrows and `Enter` belong to whichever checklist row is open." Delete the sentence "The compact side panel labels the order as priority order, reserves five band rows, and renders the five-row page containing the selected band; …" through "…adds no independent scroll position or saved UI state." and the `ACTION` column paragraph, replacing them with the chip paragraph. Add the hovered-tile tint and tooltip sentence to "Reachability overlay".

- [ ] **Step 2: §"Top-down terrain: one grid-aligned cached surface" (~line 6616)**

Replace "each tile owns an 8 × 8 cell … The resulting 768 × 512 terrain layer leaves a persistent lower strip for research and selected-band reports." with: "each tile owns an 8 × 8 cell in the overview camera and draws a 7.6 × 7.6 colored rectangle at the cell's upper-left corner. The map area is 864 × 626 logical pixels (x 20–884, y 74–700); the terrain layer occupies its top-left 768 × 512 in overview. A two-state camera (below) may show the grid at 3× within the same area." Replace the sentence beginning "There are no side walls, lighting, depth targets, 3D camera, orbit/pan/zoom controls…" with "There are no side walls, lighting, depth targets, 3D camera, free pan, or terrain-detail modes; the only view change is the uniform two-state camera, which preserves both properties below." Append a `#### Two-state camera` subsection with spec §6 verbatim (modes, trigger, override, transition ticks, clamping, nearest-neighbour scaling, clipping to the map area above the drawer).

- [ ] **Step 3: §"Field Notes: context, abstraction, and hints" (~line 7052)**

Replace the first paragraph's "docked along the lower edge of the gameplay HUD. It has a capped responsive height and its own scroll position, and it must not cover the top bar, selected-band controls, tile inspector, or required alerts. The top bar has a book-button toggle" with "docked over the lower edge of the map area in one of three states: hidden (edge tab only), compact (102 logical px), or expanded (300 logical px). The chosen height is a local UI preference. It never covers the top bar, the timeline rail, or the right panel. The drawer's edge tab has hide and more/less controls". Keep the `F` sentence and add "; Shift+F toggles compact and expanded." Add: "Below the note body the drawer lists the two newest events, turn-stamped and newest first; each is clickable and focuses that event kind's entry." Everything else in the section is unchanged.

- [ ] **Step 4: Appendix C rows (table around line 9744)**

Replace:

```markdown
| Top-down map rectangle                     | origin `(20, 74)`; `96 × 64` cells of `8 × 8` logical pixels  | Locked                                          | §8    |
| Top-down drawn tile extent                 | `7.6 × 7.6` logical pixels within each cell                    | Locked                                          | §8    |
```

with:

```markdown
| Top-down map area                          | origin `(20, 74)`; `864 × 626` logical pixels; `96 × 64` grid  | Locked                                          | §8    |
| Overview cell and drawn tile extent        | `8 × 8` cell; `7.6 × 7.6` drawn                                 | Locked                                          | §8    |
| Focus camera scale and transition          | `3×`; `15` update ticks; clamped to the map area above the drawer | Policy                                        | §8    |
| Field Notes drawer heights                 | compact `102`, expanded `300` logical px                        | Policy                                          | §8    |
| Liveability tiers (presentation only)      | food red `< RequiredFU`, amber `< 1.5 × RequiredFU`; water red `< 0.25 cap`, amber `< 0.5 cap`; degradation amber `≥ 0.25`, red `≥ 0.5`; mortality amber `≥ 0.004`, red `≥ 0.008`; shelter amber `< 0.3`; archaic present amber | Initial | §8 |
| UI settings schema                         | `2`: `FieldNotesVisible`, `MasterVolume`, `Muted`, `GuideDismissed`, `FieldNotesExpanded`; schema 1 decodes with the new fields false | Policy | §8 |
```

- [ ] **Step 5: Architecture, boundaries, dependencies**

- In the `### Package layout` block, under `pkg/render/`, change `map.go` to "scalable top-down terrain, fog, camera, picking, markers, routes, and viewport transforms", delete the `tile_info.go`, `band_window.go`, and `outcome.go` lines, add `camera.go  two-state camera geometry`. Add a `pkg/hud/` block after `pkg/ui/`:

```
pkg/hud/                 EBITENUI CHROME ADAPTER — gameapi + ui + render + ebitenui; no application/domain
  state.go, intent.go    derived State in, Intent out; comparable so the tree rebuilds only on change
  theme.go               palette, DIP→px, scaled font faces
  fixed_layout.go        absolute DIP-rect layout for the chrome regions
  panel.go               Panel lifecycle; header.go chips.go details.go move_row.go research_row.go workforce_row.go end_turn.go
  drawer.go              Field Notes drawer (hidden/compact/expanded) with clickable events
  guide_card.go          first-turn guide card
  overlays.go, end_scene.go  title/menu/storage/settings windows and the New Campaign button
```

  Under `pkg/ui/` add `bands.go`, `checklist.go`, `liveability.go`, `guide.go` one-liners.
- In `## 5. Enforced package boundaries`, update the embedded YAML to match `.golangci.yml` exactly (new `hud-adapter-dependencies` rule; the two new allow entries in the host rule).
- In the §3 table row "Historical/scientific explanation has no persistent UI home", change "docked" wording to the drawer.
- In Appendix A, add a bullet: "**ebitenui v0.7.3.** Widgets lay out and hit-test in the coordinates of the image they draw to; the chrome therefore lays out in render pixels with sizes multiplied by the presentation scale. Measured on the dependency skeleton at +169 KB Brotli over Ebitengine alone, against a 540 KB gap under the live ceiling."
- Add a `### Keyboard reference` subsection at the end of "Gameplay stats layout" with the spec §8 global and row-owned key tables, noting the removed `W`, `[`, `]`, and gameplay volume keys, and stating that the drawer scrolls with the wheel only (Shift+PgUp/PgDn from the spec is not bound; the widget library has no scroll setter).

- [ ] **Step 6: Verify and commit**

Run: `go run ./tools/check_audits` (unaffected but cheap) and `grep -n 'five band rows\|ACTION\b\|lower strip\|orbit/pan/zoom\|book-button' docs/DESIGN.md` — Expected: no matches remain except in historical decision-table cells that describe the old problem statement.

```bash
git add docs/DESIGN.md
git commit -m "DESIGN.md: task-oriented panel, drawer, camera, guide, and ebitenui boundary

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

### Task 18: Final verification

**Files:** none new.

- [ ] **Step 1: Full native check**

```bash
go build ./... && go vet ./... && go test ./... && golangci-lint run && go run ./tools/check_audits
```

Expected: all green.

- [ ] **Step 2: Web build, size gate, smoke**

```bash
./build_web.sh --release && ./tools/check_wasm_size.sh web/main.wasm /tmp/wasm-size.json && cat /tmp/wasm-size.json
```

Expected: brotli size at or under `3600000`; the recorded number should be roughly 3.06 MB + 0.17 MB. If `wasm-opt` is unavailable locally, use `./build_web.sh --dev` for the smoke test and let CI measure size.

```bash
npm --prefix tools/web-e2e test
```

Expected: exit 0 (the smoke test's keyboard sequence is unchanged).

- [ ] **Step 3: Visual pass**

```bash
go run . -screenshot /tmp/hud-final-turn0.png -turns 0
go run . -screenshot /tmp/hud-final-turn12.png -turns 12
```

Open both. Compare against the mockups in `.superpowers/brainstorm/*/content/layout-v4-turn0.html` and `layout-v5-move.html`: chips colored, Move row open with HERE/TARGET, End turn amber with the count, drawer over the map bottom, guide card at turn 0, no bottom strip, focus camera centred on the selected band. On a 2× display confirm text is crisp and the panel edge lands at logical x = 908.

- [ ] **Step 4: Interactive pass** (`go run .`)

Title → Continue with the mouse. Click chips, open each row by header, hover tiles and confirm the tooltip/tint and the TARGET column, click Move here, choose a tech by clicking, drag a workforce slider and use −/+, Apply, click End turn twice (soft block), Esc → menu → Settings slider → Show first-turn guide → Back. Keyboard: Tab, PgUp/PgDn, arrows in each row, Enter, Space, Z, ?, F, Shift+F, Shift+PgDn.

- [ ] **Step 5: Finish the branch**

Follow `superpowers:finishing-a-development-branch`: open a PR from `hud-redesign` to `main` with a summary of the four phases and the spec link, ending with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.

---

## Self-review notes (kept for executors)

- Spec §3.1's list of functions moving to `pkg/ui` is realised by Tasks 2–4; `campaignEraLabel`/`macroWarningLabel` live in `pkg/hud/header.go` instead of `pkg/ui` because only the header reads them.
- Spec §4's "`+N` chip opens the full attention-ordered list" is `IntentToggleBandList` + `State.BandListOpen` (Task 8).
- Spec §8's shortcut sheet is `IntentToggleShortcuts` + `State.ShortcutsOpen` (Tasks 12–13).
- Spec §10's "synthetic cursor updater" test is realised with `Button.Click()` on held widget handles, which exercises the same click handlers without needing a real cursor.
- Every removed hotkey and every new one is listed in Task 12; the smoke test's keys are all preserved.
- Deviation from spec §8: Shift+PgUp/PgDn drawer scrolling is not bound because ebitenui's `TextArea` has no programmatic scroll; the drawer scrolls with the wheel. Task 17 records this in DESIGN.md.
