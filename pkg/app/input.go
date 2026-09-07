package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// gameplayKeysActive reports whether handleGameplayKeys should process
// gameplay keys. It is false while the modal shortcut sheet is open: the
// scene stays SceneGameplay while the sheet is shown, so without this guard
// Space/Tab/etc. would fire behind a sheet that claims only ? or Esc closes
// it. Esc itself is handled earlier in Update, via escape(), which peels
// shortcutsOpen before handleGameplayKeys ever runs; only the ? (Shift+/)
// toggle needs to keep working from inside this function while the guard is
// closed. Factored out as its own method (rather than inlined into the
// switch below) because Ebitengine key state cannot be injected in tests —
// this predicate is the seam tests can exercise directly.
func (g *Game) gameplayKeysActive() bool {
	return !g.shortcutsOpen
}

// handleGameplayKeys is the keyboard half of spec §8: global keys first, then
// the keys the open checklist row owns.
func (g *Game) handleGameplayKeys() {
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	if !g.gameplayKeysActive() {
		if inpututil.IsKeyJustPressed(ebiten.KeySlash) && shift {
			g.toggleShortcutSheet()
		}
		return
	}
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
		g.toggleCameraFocus()
	case inpututil.IsKeyJustPressed(ebiten.KeySlash) && shift:
		g.toggleShortcutSheet()
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		g.changeOpenRow(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		g.changeOpenRow(1)
	case shift && inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
		g.changeOpenRow(-1)
	case shift && inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
		g.changeOpenRow(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyD) && g.openRow != ui.RowWorkforce:
		g.toggleDetails()
	case inpututil.IsKeyJustPressed(splitBandHotkey):
		g.splitSelectedBand()
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		g.requestInterbreed()
	case inpututil.IsKeyJustPressed(ebiten.KeyJ):
		g.selectNextInterbreedTarget()
	case inpututil.IsKeyJustPressed(ebiten.KeyG):
		g.focusNextTraitNote()
	case inpututil.IsKeyJustPressed(ebiten.KeyB):
		g.moveToBestTile()
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.endTurn(true)
	default:
		for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if inpututil.IsKeyJustPressed(key) {
				g.chooseResearchTechnology(gameapi.Tech(index))
				return
			}
		}
		for _, key := range [...]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyEnter, ebiten.KeyMinus, ebiten.KeyEqual, ebiten.KeyA, ebiten.KeyD} {
			if inpututil.IsKeyJustPressed(key) {
				g.handleRowKey(key, shift)
				return
			}
		}
	}
}

// handleRowKey routes arrows, Enter, and Workforce's −/+ to whichever row is
// open; none of them act while a different row owns the keyboard.
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
		}
	case ui.RowResearch:
		switch key {
		case ebiten.KeyArrowUp:
			g.researchCursor = (g.researchCursor + gameapi.TechCount - 1) % gameapi.TechCount
		case ebiten.KeyArrowDown:
			g.researchCursor = (g.researchCursor + 1) % gameapi.TechCount
		case ebiten.KeyEnter:
			g.chooseResearchTechnology(g.researchCursor)
		}
	case ui.RowWorkforce:
		switch key {
		case ebiten.KeyArrowUp:
			g.assignmentRole = (g.assignmentRole + gameapi.AssignmentCount - 1) % gameapi.AssignmentCount
		case ebiten.KeyArrowDown:
			g.assignmentRole = (g.assignmentRole + 1) % gameapi.AssignmentCount
		case ebiten.KeyArrowLeft, ebiten.KeyMinus:
			g.adjustSelectedRole(-100, shift)
		case ebiten.KeyArrowRight, ebiten.KeyEqual:
			g.adjustSelectedRole(100, shift)
		case ebiten.KeyEnter, ebiten.KeyA:
			g.applyAssignmentDraft()
		case ebiten.KeyD:
			g.discardAssignmentDraft()
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

// toggleShortcutSheet flips the modal shortcut sheet (spec §8), drawn by the
// panel's overlay window. Keyboard (Shift+/) and the panel's
// IntentToggleShortcuts share this one path so they cannot disagree.
func (g *Game) toggleShortcutSheet() {
	g.shortcutsOpen = !g.shortcutsOpen
}
