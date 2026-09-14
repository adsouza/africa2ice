package ui

import "github.com/adsouza/africa2ice/pkg/gameapi"

// AssignmentDraft owns the player's uncommitted workforce edits. It never
// rebalances another role, sends a command, or changes a published frame.
type AssignmentDraft struct {
	allocation [gameapi.AssignmentCount]uint16
	baseline   [gameapi.AssignmentCount]uint16
	bandID     gameapi.BandID
	visible    bool
	Role       gameapi.WorkforceRole
}

func (draft AssignmentDraft) Allocation() [gameapi.AssignmentCount]uint16 { return draft.allocation }
func (draft AssignmentDraft) BandID() gameapi.BandID                      { return draft.bandID }
func (draft AssignmentDraft) Visible() bool                               { return draft.visible }
func (draft AssignmentDraft) Dirty() bool                                 { return draft.visible && draft.allocation != draft.baseline }

func (draft AssignmentDraft) Valid() bool {
	if !draft.visible {
		return false
	}
	var total uint32
	for _, points := range draft.allocation {
		total += uint32(points)
	}
	return total == 10_000
}

// Sync preserves a dirty draft when the same band is projected again. The host
// uses force only after an accepted apply, load, or deliberate selection change.
func (draft *AssignmentDraft) Sync(band *gameapi.Band, force bool) {
	if band == nil || band.Species != gameapi.HomoSapiens {
		draft.Clear()
		return
	}
	if !force && draft.visible && draft.bandID == band.ID && draft.Dirty() {
		return
	}
	draft.bandID = band.ID
	draft.allocation, draft.baseline = band.AllocationBP, band.AllocationBP
	draft.visible = true
	if draft.Role >= gameapi.AssignmentCount {
		draft.Role = gameapi.Foraging
	}
}

func (draft *AssignmentDraft) Clear() { draft.visible = false }

// Edit reports whether the resulting draft is dirty, so the host can explain
// the Apply/Discard choices. Only the selected role is clamped or changed.
func (draft *AssignmentDraft) Edit(delta int) bool {
	if !draft.visible || draft.Role >= gameapi.AssignmentCount {
		return false
	}
	value := int(draft.allocation[draft.Role]) + delta
	draft.allocation[draft.Role] = uint16(min(max(value, 0), 10_000))
	return draft.Dirty()
}

func (draft *AssignmentDraft) Discard() bool {
	if !draft.Dirty() {
		return false
	}
	draft.allocation = draft.baseline
	return true
}

// Command returns a value copy; rejection leaves the draft available to edit.
func (draft AssignmentDraft) Command() (gameapi.SetAssignment, bool) {
	if !draft.Dirty() || !draft.Valid() {
		return gameapi.SetAssignment{}, false
	}
	return gameapi.SetAssignment{BandID: draft.bandID, AllocationBP: draft.allocation}, true
}
