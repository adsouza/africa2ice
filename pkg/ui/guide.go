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
// predicate is already satisfied, looping so a player who does both out of
// order (choosing Research before Move, say) catches the guide up to
// Workforce in one call rather than needing a second Observe to notice the
// step Next already skipped past. Workforce is optional and advances only on
// Next, so the loop always stops there.
func (guide GuideState) Observe(band *gameapi.Band) GuideState {
	if band == nil {
		return guide
	}
	for {
		switch {
		case guide.Step == GuideMove && MoveDone(*band):
			guide = guide.Next()
		case guide.Step == GuideResearch && ResearchDone(*band):
			guide = guide.Next()
		default:
			return guide
		}
	}
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
		return "Each turn, every band may make one move. This band's reachable tiles are outlined on the map; the gold one has the best food. Click it, or click Best tile below. Staying put is also fine."
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
