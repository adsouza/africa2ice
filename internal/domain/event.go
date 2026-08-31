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
	EventKindCount
)

// Event is bounded historical context. It is display state only and never an
// input to simulation decisions.
type Event struct {
	Turn    int
	Kind    EventKind
	BandID  BandID
	TileID  TileID
	Region  Region
	Summary string
}

func eventKindName(kind AcuteKind) string {
	switch kind {
	case AcutePredation:
		return "predation"
	case AcuteDiseaseOutbreak:
		return "disease outbreak"
	case AcuteFloodStorm:
		return "flood or storm"
	case AcuteExposureFall:
		return "exposure or fall"
	case AcuteCrossingMishap:
		return "crossing mishap"
	default:
		return "acute"
	}
}

func technologyName(technology Technology) string {
	if technology >= TechCount {
		return "unknown technology"
	}
	return [...]string{"Firecraft", "Hafted Tools", "Plant Knowledge", "Tailored Clothing", "Cordage and Nets", "Campcraft", "Medicinal Knowledge", "Trapping", "Coastal Navigation"}[technology]
}

func (world *World) appendEvent(event Event) {
	world.events = append(world.events, event)
	if overflow := len(world.events) - MaxEvents; overflow > 0 {
		copy(world.events, world.events[overflow:])
		world.events = world.events[:MaxEvents]
	}
}
