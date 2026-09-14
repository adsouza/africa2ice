package application

import (
	"fmt"

	"github.com/adsouza/africa2ice/internal/domain"
)

// eventSummary is presentation wording for historical facts. Legacy save text
// stays verbatim because those saves did not record the facts needed to reword it.
func eventSummary(event domain.Event) string {
	if event.LegacySummary != "" {
		return event.LegacySummary
	}
	switch event.Kind {
	case domain.EventMigration:
		return fmt.Sprintf("Band %d migrated.", event.BandID)
	case domain.EventSplit:
		return fmt.Sprintf("Band %d split from band %d.", event.BandID, event.Details.ParentBandID)
	case domain.EventTechnology:
		return fmt.Sprintf("Band %d learned %s.", event.BandID, mapTech(event.Details.Technology))
	case domain.EventInterbreeding:
		return fmt.Sprintf("Band %d interbred with an archaic band.", event.BandID)
	case domain.EventAcuteIncident:
		return fmt.Sprintf("Band %d suffered %s.", event.BandID, acuteIncidentPhrase(event.Details.AcuteKind))
	case domain.EventMacroEpisode:
		return fmt.Sprintf("Band %d was affected by the Campanian eruption.", event.BandID)
	case domain.EventAchievement:
		return fmt.Sprintf("Sapiens established %s.", mapRegion(event.Region))
	case domain.EventExtinction:
		label := fmt.Sprintf("Band %d", event.BandID)
		if event.Details.Species == domain.ArchaicHominin {
			label = fmt.Sprintf("Archaic band %d", event.BandID)
		}
		cause := ""
		if event.Details.MortalityCause < domain.MortalityCauseCount {
			cause = [...]string{"", "starvation", "seasonal mortality", "chronic mortality", "a macro episode", "an acute incident"}[event.Details.MortalityCause]
		}
		if cause != "" {
			return fmt.Sprintf("%s died out after %s.", label, cause)
		}
		return label + " died out."
	default:
		return ""
	}
}

func acuteIncidentPhrase(kind domain.AcuteKind) string {
	name := "acute"
	if kind < domain.AcuteKindCount {
		name = [...]string{"predation", "disease outbreak", "flood or storm", "exposure or fall", "crossing mishap"}[kind]
	}
	article := "a"
	switch name[0] {
	case 'a', 'e', 'i', 'o', 'u':
		article = "an"
	}
	return article + " " + name + " incident"
}
