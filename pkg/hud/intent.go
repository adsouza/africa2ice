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
	IntentToggleBandList
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
