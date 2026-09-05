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
	IntentAdjustRole
	IntentApplyWorkforce
	IntentDiscardWorkforce
	IntentEndTurn
	IntentGuideNext
	IntentGuideDismiss
	IntentCameraToggle
	IntentFocusTrait
	IntentFocusEvent
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
	IntentToggleReducedMotion
	IntentShowGuide
	IntentToggleBandList
	IntentToggleShortcuts
	IntentKindCount
)

// String returns a stable, lowercase, hyphenated name for the kind: the
// vocabulary the UI observability log (internal/adapters/logging) speaks.
// Every value below IntentKindCount must return a distinct, non-empty name
// (see TestIntentKindStringIsDistinctAndNonEmpty) so a log reader can always
// tell one intent kind from another.
func (kind IntentKind) String() string {
	switch kind {
	case IntentNone:
		return "none"
	case IntentSelectBand:
		return "select-band"
	case IntentOpenRow:
		return "open-row"
	case IntentToggleDetails:
		return "toggle-details"
	case IntentSetNotesMode:
		return "set-notes-mode"
	case IntentMoveTo:
		return "move-to"
	case IntentMoveToBest:
		return "move-to-best"
	case IntentSplit:
		return "split"
	case IntentInterbreed:
		return "interbreed"
	case IntentChooseResearch:
		return "choose-research"
	case IntentAdjustRole:
		return "adjust-role"
	case IntentApplyWorkforce:
		return "apply-workforce"
	case IntentDiscardWorkforce:
		return "discard-workforce"
	case IntentEndTurn:
		return "end-turn"
	case IntentGuideNext:
		return "guide-next"
	case IntentGuideDismiss:
		return "guide-dismiss"
	case IntentCameraToggle:
		return "camera-toggle"
	case IntentFocusTrait:
		return "focus-trait"
	case IntentFocusEvent:
		return "focus-event"
	case IntentOpenMenu:
		return "open-menu"
	case IntentBack:
		return "back"
	case IntentContinue:
		return "continue"
	case IntentNewCampaign:
		return "new-campaign"
	case IntentOpenStorage:
		return "open-storage"
	case IntentOpenSettings:
		return "open-settings"
	case IntentReturnToTitle:
		return "return-to-title"
	case IntentSaveSlot:
		return "save-slot"
	case IntentLoadSlot:
		return "load-slot"
	case IntentDeleteSlot:
		return "delete-slot"
	case IntentSetVolume:
		return "set-volume"
	case IntentToggleMute:
		return "toggle-mute"
	case IntentToggleReducedMotion:
		return "toggle-reduced-motion"
	case IntentShowGuide:
		return "show-guide"
	case IntentToggleBandList:
		return "toggle-band-list"
	case IntentToggleShortcuts:
		return "toggle-shortcuts"
	default:
		return "unknown"
	}
}

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
