package domain

import "fmt"

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

// acuteIncidentPhrase names an acute hazard as a noun phrase for the event
// feed. The article is chosen from the leading letter, which is exact over
// this closed vocabulary — every entry is an ordinary consonant- or
// vowel-initial word — rather than a general English rule.
func acuteIncidentPhrase(kind AcuteKind) string {
	name := eventKindName(kind)
	article := "a"
	switch name[0] {
	case 'a', 'e', 'i', 'o', 'u':
		article = "an"
	}
	return article + " " + name + " incident"
}

// mortalityCause names the heaviest component of a band's last mortality
// report, for the clause explaining why it died out. Ties resolve in field
// order, so a report always yields the same sentence. The two generic phrases
// reuse the feed's own kind labels — the specific hazard already arrives on
// the Acute Incident or Macro Episode line the same turn. An empty result
// means nothing was recorded and the caller should omit the clause.
func mortalityCause(report MortalityReport) string {
	causes := []struct {
		toll float64
		name string
	}{
		{report.Starvation, "starvation"},
		{report.Seasonal, "seasonal mortality"},
		{report.Chronic, "chronic mortality"},
		{report.Macro, "a macro episode"},
		{report.Acute, "an acute incident"},
	}
	heaviest := ""
	worst := 0.0
	for _, cause := range causes {
		if cause.toll > worst {
			worst, heaviest = cause.toll, cause.name
		}
	}
	return heaviest
}

// extinctionSummary is the feed line for a band the turn removed. Archaic
// bands are prefixed because every death in an automated campaign is archaic,
// and an undifferentiated line would read as the player losing their own.
func extinctionSummary(band Band) string {
	label := fmt.Sprintf("Band %d", band.ID)
	if band.Species == ArchaicHominin {
		label = fmt.Sprintf("Archaic band %d", band.ID)
	}
	if cause := mortalityCause(band.LastMortality); cause != "" {
		return fmt.Sprintf("%s died out after %s.", label, cause)
	}
	return label + " died out."
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
