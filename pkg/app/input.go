package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// gameplayKeysActive blocks commands behind the shortcut sheet. Escape is
// handled by the scene first; Shift+/ can still close the sheet here.
func (g *Game) gameplayKeysActive() bool {
	return !g.shortcutsOpen
}

// handleGameplayKeys is the keyboard half of spec §8: global keys first, then
// the keys the open checklist row owns.
func (g *Game) handleGameplayKeys() {
	g.handleGameplayKeyState(ebiten.IsKeyPressed, inpututil.IsKeyJustPressed)
}

// Keep device polling at the edge so key priority and modal routing can be
// exercised with the same input state in tests.
func (g *Game) handleGameplayKeyState(pressed, justPressed func(ebiten.Key) bool) {
	shift := pressed(ebiten.KeyShift)
	if !g.gameplayKeysActive() {
		if justPressed(ebiten.KeySlash) && shift {
			g.toggleShortcutSheet()
		}
		return
	}
	switch {
	case justPressed(ebiten.KeyTab):
		if g.workforce.Dirty() {
			g.showNotice("Apply or discard workforce changes")
		} else {
			g.clearMigrationPreview()
			if shift {
				g.selectPreviousSapiens()
			} else {
				g.selectNextSapiens()
			}
		}
	case justPressed(fieldNotesHotkey):
		if shift {
			g.toggleNotesExpanded()
		} else {
			g.toggleFieldNotes()
		}
	case justPressed(ebiten.KeyM):
		g.toggleMute()
	case justPressed(ebiten.KeyZ):
		g.toggleCameraFocus()
	case justPressed(ebiten.KeySlash) && shift:
		g.toggleShortcutSheet()
	case justPressed(ebiten.KeyPageUp):
		g.changeOpenRow(-1)
	case justPressed(ebiten.KeyPageDown):
		g.changeOpenRow(1)
	case shift && justPressed(ebiten.KeyArrowUp):
		g.changeOpenRow(-1)
	case shift && justPressed(ebiten.KeyArrowDown):
		g.changeOpenRow(1)
	case justPressed(ebiten.KeyD) && g.openRow != ui.RowWorkforce:
		g.toggleDetails()
	case justPressed(splitBandHotkey):
		g.splitSelectedBand()
	case justPressed(ebiten.KeyI):
		g.requestInterbreed()
	case justPressed(ebiten.KeyJ):
		g.selectNextInterbreedTarget()
	case justPressed(ebiten.KeyG):
		g.focusNextTraitNote()
	case justPressed(ebiten.KeyB):
		g.moveToBestTile()
	case justPressed(ebiten.KeySpace):
		g.endTurn(true)
	default:
		for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if justPressed(key) {
				g.chooseResearchTechnology(gameapi.Tech(index))
				return
			}
		}
		for _, key := range [...]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyEnter, ebiten.KeyMinus, ebiten.KeyEqual, ebiten.KeyA, ebiten.KeyD} {
			if justPressed(key) {
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
			g.workforce.Role = (g.workforce.Role + gameapi.AssignmentCount - 1) % gameapi.AssignmentCount
		case ebiten.KeyArrowDown:
			g.workforce.Role = (g.workforce.Role + 1) % gameapi.AssignmentCount
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
