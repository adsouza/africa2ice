package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// panelDisclosure is the HUD panel's open/closed state: which checklist row
// is open and whether the player chose it, plus the details, band-list, and
// end-turn-confirmation toggles. A new selection, a load, a new campaign, or a
// completed turn resets all of it (spec §5.2).
type panelDisclosure struct {
	openRow      ui.ChecklistRow
	rowChosen    bool // player opened a row explicitly; auto-advance yields until reset
	detailsOpen  bool
	bandListOpen bool
	endTurnArmed bool
}

func (d *panelDisclosure) reset(band *gameapi.Band) {
	*d = panelDisclosure{openRow: ui.DefaultOpenRow(band)}
}

// advance moves to the next unfinished row after an accepted action unless
// the player chose a row explicitly.
func (d *panelDisclosure) advance(band *gameapi.Band) {
	if !d.rowChosen {
		d.openRow = ui.DefaultOpenRow(band)
	}
}

func (d *panelDisclosure) choose(row ui.ChecklistRow) { d.openRow, d.rowChosen = row, true }

// step moves the accordion by delta rows, wrapping, as a player choice.
func (d *panelDisclosure) step(delta int) {
	count := int(ui.ChecklistRowCount)
	d.choose(ui.ChecklistRow((int(d.openRow) + delta + count) % count))
}

// confirmEndTurn reports whether an end-turn request may proceed. While bands
// still wait to move, the first unforced request only arms the confirmation.
func (d *panelDisclosure) confirmEndTurn(waiting int, force bool) bool {
	if waiting > 0 && !force && !d.endTurnArmed {
		d.endTurnArmed = true
		return false
	}
	d.endTurnArmed = false
	return true
}
