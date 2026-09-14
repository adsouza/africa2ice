package domain

const MaxEvents = 128

type EventKind uint8

const (
	EventMigration EventKind = iota
	EventSplit
	EventTechnology
	EventInterbreeding
	EventAcuteIncident
	EventMacroEpisode
	EventAchievement
	EventExtinction
	EventKindCount
)

// Event is bounded historical context. It is display state only and never an
// input to simulation decisions.
type Event struct {
	Turn   int
	Kind   EventKind
	BandID BandID
	TileID TileID
	Region Region
	// LegacySummary preserves historical text from version-1 saves.
	// New events contain facts only.
	LegacySummary string
	Details       EventDetails
}

type MortalityCause uint8

const (
	MortalityUnspecified MortalityCause = iota
	MortalityStarvation
	MortalitySeasonal
	MortalityChronic
	MortalityMacro
	MortalityAcute
	MortalityCauseCount
)

// EventDetails is a value so copying an event cannot alias mutable facts.
type EventDetails struct {
	ParentBandID   BandID
	Technology     Technology
	AcuteKind      AcuteKind
	Species        Species
	MortalityCause MortalityCause
}

func mortalityCause(report MortalityReport) MortalityCause {
	cause := MortalityUnspecified
	worst := 0.0
	for index, toll := range [...]float64{report.Starvation, report.Seasonal, report.Chronic, report.Macro, report.Acute} {
		if toll > worst {
			worst, cause = toll, MortalityCause(index+1)
		}
	}
	return cause
}

func (event Event) validDetails(nextBandID BandID) bool {
	if event.LegacySummary != "" {
		return len(event.LegacySummary) <= 256 && event.Details == (EventDetails{})
	}
	d := event.Details
	expected := EventDetails{}
	switch event.Kind {
	case EventSplit:
		if d.ParentBandID == 0 || d.ParentBandID >= nextBandID || d.ParentBandID == event.BandID {
			return false
		}
		expected.ParentBandID = d.ParentBandID
	case EventTechnology:
		if d.Technology >= TechCount {
			return false
		}
		expected.Technology = d.Technology
	case EventAcuteIncident:
		if d.AcuteKind >= AcuteKindCount {
			return false
		}
		expected.AcuteKind = d.AcuteKind
	case EventExtinction:
		if d.Species > ArchaicHominin || d.MortalityCause >= MortalityCauseCount {
			return false
		}
		expected.Species, expected.MortalityCause = d.Species, d.MortalityCause
	}
	return d == expected
}

// appendBandEvent records an event about a specific band, dropping the
// computer's routine activity before it reaches the feed. Archaic bands move,
// research, split and suffer hazards every turn, and there are dozens of them:
// measured at turn 200 of a reference campaign, all 128 retained events were
// archaic and 125 were migrations, so not one of the player's own events
// survived the window. Milestones (extinction, macro episodes) and anything a
// sapiens band is party to (interbreeding) still surface for archaic bands.
//
// Every band-scoped emission goes through here rather than testing the species
// at each call site, so a later event kind inherits the policy instead of
// having to remember it.
func (world *World) appendBandEvent(band Band, event Event) {
	if band.Species == ArchaicHominin && archaicBackgroundNoise(event.Kind) {
		return
	}
	world.appendEvent(event)
}

func archaicBackgroundNoise(kind EventKind) bool {
	switch kind {
	case EventMigration, EventTechnology, EventSplit, EventAcuteIncident:
		return true
	default:
		return false
	}
}

func (world *World) appendEvent(event Event) {
	world.events = append(world.events, event)
	if overflow := len(world.events) - MaxEvents; overflow > 0 {
		copy(world.events, world.events[overflow:])
		world.events = world.events[:MaxEvents]
	}
}
