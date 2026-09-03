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
		g.toggleShortcutSheet()
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
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		g.requestInterbreed()
	case inpututil.IsKeyJustPressed(ebiten.KeyJ):
		g.selectNextInterbreedTarget()
	case inpututil.IsKeyJustPressed(ebiten.KeyG):
		g.focusNextTraitNote()
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.endTurn(true)
	case inpututil.IsKeyJustPressed(ebiten.KeyMinus):
		g.adjustSelectedRole(-100, shift)
	case inpututil.IsKeyJustPressed(ebiten.KeyEqual):
		g.adjustSelectedRole(100, shift)
	default:
		for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if inpututil.IsKeyJustPressed(key) {
				g.chooseResearchTechnology(gameapi.Tech(index))
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

// toggleShortcutSheet flips the modal shortcut sheet (spec §8); Task 13 draws
// it. Keyboard (Shift+/) and the panel's IntentToggleShortcuts share this one
// path so they cannot disagree.
func (g *Game) toggleShortcutSheet() {
	g.shortcutsOpen = !g.shortcutsOpen
}

// toggleCameraOverride is a stub; Task 15 fills it in.
func (g *Game) toggleCameraOverride() {}
