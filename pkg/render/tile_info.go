package render

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const archaicPresenceLineIndex = 6

type mapLegendEntry struct {
	label   string
	meaning string
	color   color.RGBA
}

func mapLegendEntries(aridity float64) [8]mapLegendEntry {
	return [8]mapLegendEntry{
		{label: "Riverine woodland", meaning: "plant food; disease", color: climateBiomeColor(gameapi.RiverineWoodland, aridity)},
		{label: "Savanna", meaning: "mixed food; drought", color: climateBiomeColor(gameapi.Savanna, aridity)},
		{label: "Coastal shrubland", meaning: "aquatic food; storms", color: climateBiomeColor(gameapi.CoastalShrubland, aridity)},
		{label: "Mountain highlands", meaning: "low capacity; cold/falls", color: climateBiomeColor(gameapi.MountainousHighlands, aridity)},
		{label: "Semi-arid desert", meaning: "little food/water; heat", color: climateBiomeColor(gameapi.SemiAridDesert, aridity)},
		{label: "Glacial tundra", meaning: "little plant food; freezing", color: climateBiomeColor(gameapi.GlacialTundra, aridity)},
		{label: "Water", meaning: "cannot be occupied", color: waterTileColor},
		{label: "Unknown", meaning: "not yet explored", color: unexploredTileColor},
	}
}

type tileLiveabilitySummary struct {
	heading            string
	status             string
	biome              string
	region             string
	foodStock          float64
	foodCap            float64
	floraStock         float64
	faunaStock         float64
	waterStock         float64
	waterCap           float64
	ecologicalK        float64
	degradation        float64
	seasonalRisk       float64
	chronicRisk        float64
	crowdingDecline    float64
	hasRisk            bool
	temperatureC       float64
	naturalShelter     float64
	movementCost       float64
	visibleMacroImpact gameapi.MacroImpactSummary
	archaicBandCount   int
	archaicPopulation  uint64
	showDetails        bool
}

func currentTileSummary(frame *gameapi.Frame, band *gameapi.Band) tileLiveabilitySummary {
	if band == nil {
		return tileLiveabilitySummary{heading: "HERE", status: "No active band"}
	}
	summary := summarizeTile(frame, band.TileID, "HERE")
	summary.status = "current"
	summary.seasonalRisk = band.SeasonalMortalityRate
	summary.chronicRisk = band.ChronicMortalityRate
	summary.hasRisk = true
	return summary
}

func targetTileSummary(frame *gameapi.Frame, band *gameapi.Band, preview MigrationPreview) tileLiveabilitySummary {
	if band == nil {
		return tileLiveabilitySummary{heading: "TARGET", status: "No active band"}
	}
	tileID, status, available := gameapi.TileID(0), "Use arrow keys", false
	if preview.Visible && preview.BandID == band.ID {
		tileID, status, available = preview.TileID, "arrow cursor", true
	} else if band.HasQueuedMigration {
		tileID, status, available = band.QueuedMigration, "queued", true
	}
	if !available {
		return tileLiveabilitySummary{heading: "TARGET", status: status}
	}
	summary := summarizeTile(frame, tileID, "TARGET")
	if !summary.showDetails {
		return summary
	}
	if candidate, ok := migrationCandidate(*band, tileID); ok {
		summary.status = status + " · reachable"
		summary.seasonalRisk = candidate.SeasonalMortalityRate
		summary.chronicRisk = candidate.ChronicMortalityRate
		summary.crowdingDecline = candidate.CrowdingDecline
		summary.hasRisk = true
	} else {
		summary.status = status + " · not reachable"
	}
	return summary
}

func summarizeTile(frame *gameapi.Frame, tileID gameapi.TileID, heading string) tileLiveabilitySummary {
	summary := tileLiveabilitySummary{heading: heading, status: "Invalid tile"}
	if frame == nil || int(tileID) >= len(frame.Tiles) {
		return summary
	}
	tile := frame.Tiles[tileID]
	if !tile.Explored {
		summary.status = "Unexplored · details hidden"
		return summary
	}
	if !tile.Land {
		summary.status = "Open water · cannot occupy"
		return summary
	}
	if tile.BaselineK <= 0 {
		summary.status = "Uninhabitable in this climate"
		return summary
	}
	summary.showDetails = true
	summary.biome = tile.Biome.String()
	summary.region = tile.Region.String()
	summary.foodStock = tile.FloraStock + tile.FaunaStock
	summary.foodCap = tile.FloraCap + tile.FaunaCap
	summary.floraStock = tile.FloraStock
	summary.faunaStock = tile.FaunaStock
	summary.waterStock = tile.WaterStock
	summary.waterCap = tile.WaterCap
	summary.ecologicalK = tile.EcologicalK
	summary.degradation = tile.Degradation
	summary.temperatureC = tile.LocalTemperatureC
	summary.naturalShelter = tile.NaturalShelter
	summary.movementCost = tile.MovementCost
	summary.visibleMacroImpact = tile.VisibleMacroImpact
	for _, band := range frame.Bands {
		if band.Species != gameapi.ArchaicHominin || band.TileID != tileID || band.Population == 0 {
			continue
		}
		summary.archaicBandCount++
		summary.archaicPopulation += uint64(band.Population)
	}
	return summary
}

func migrationCandidate(band gameapi.Band, tileID gameapi.TileID) (gameapi.MigrationCandidate, bool) {
	for _, candidate := range band.MigrationCandidates {
		if candidate.TileID == tileID {
			return candidate, true
		}
	}
	return gameapi.MigrationCandidate{}, false
}

func liveabilityLines(summary tileLiveabilitySummary) [9]string {
	lines := [9]string{summary.status}
	if !summary.showDetails {
		return lines
	}
	lines[0] = summary.biome
	lines[1] = fmt.Sprintf("%s · %.0f°C", summary.region, summary.temperatureC)
	lines[2] = fmt.Sprintf("Food stock %.0f/%.0f FU", summary.foodStock, summary.foodCap)
	lines[3] = fmt.Sprintf("Plants %.0f · animals %.0f", summary.floraStock, summary.faunaStock)
	lines[4] = fmt.Sprintf("Water %.0f/%.0f WU", summary.waterStock, summary.waterCap)
	lines[5] = fmt.Sprintf("Capacity %.0f · degraded %.0f%%", summary.ecologicalK, summary.degradation*100)
	lines[archaicPresenceLineIndex] = formatArchaicPresence(summary.archaicBandCount, summary.archaicPopulation)
	switch {
	case summary.crowdingDecline > 0:
		// Crowding dwarfs the per-person hazards whenever it applies at all — in a
		// traced case by three orders of magnitude — so the line names it and
		// carries the loss in people, which is what the choice actually costs.
		// The column is about thirty-six characters wide, so the two hazard rates
		// combine rather than being dropped.
		lines[7] = fmt.Sprintf("Crowding −%.0f · hazards %.2f%%",
			summary.crowdingDecline, float64((summary.seasonalRisk+summary.chronicRisk)*100))
	case summary.hasRisk:
		lines[7] = fmt.Sprintf("Seasonal %.2f%% · chronic %.2f%%", summary.seasonalRisk*100, summary.chronicRisk*100)
	default:
		lines[7] = "Risk unavailable for route"
	}
	lines[8] = fmt.Sprintf("Shelter %.0f%% · travel ×%.2f", summary.naturalShelter*100, summary.movementCost)
	if summary.visibleMacroImpact.Visible {
		lines[8] = fmt.Sprintf("Impact food ×%.2f · K ×%.2f", summary.visibleMacroImpact.ResourceFactor, summary.visibleMacroImpact.HabitatFactor)
	}
	return lines
}

func formatArchaicPresence(bandCount int, population uint64) string {
	if bandCount == 0 {
		return "Archaic hominins: none"
	}
	bandLabel := "band"
	if bandCount != 1 {
		bandLabel = "bands"
	}
	return fmt.Sprintf("Archaic hominins: %d %s · pop %d", bandCount, bandLabel, population)
}
