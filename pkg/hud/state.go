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
	Scene ui.SceneID
	// StorageSaving distinguishes the save browser from the load browser.
	// It is data rather than an inference from StorageHeading, so renaming
	// the heading cannot silently turn saves into loads.
	StorageSaving    bool
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
	ResearchCursor  gameapi.Tech
	DetailsOpen     bool
	BandListOpen    bool
	Workforce       WorkforceDraft
	InterbreedFocus gameapi.BandID
	EndTurn         ui.EndTurnGate
	Note            render.FieldNote
	NotesMode       NotesMode
	Guide           ui.GuideState
	Camera          CameraState
	ShortcutsOpen   bool
	Overlay         OverlayState
	// CampaignOver is true once the campaign has ended (frame.CampaignResult
	// != gameapi.Ongoing): every band action control goes dead — the Move
	// row's four buttons, every research technology button, and the
	// Workforce row's sliders, −/+, Apply and Discard — while the rows keep
	// showing their data (spec §5.3, Wave I item I3). It participates in the
	// ordinary structural comparison like every other field above, so no
	// refresh path needs to know about it separately.
	CampaignOver bool
	Ending       render.EndScene
	Viewport     render.Viewport
	Transform    render.PresentationTransform
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
