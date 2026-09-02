package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

// Presentation-only liveability tiers (spec §5.4). Never simulation inputs.
const (
	foodAmberRequirementFactor = 1.5  // amber below 1.5 × last turn's RequiredFU
	waterAmberCapFraction      = 0.5  // amber below half of cap
	waterRedCapFraction        = 0.25 // red below a quarter of cap
	degradationAmber           = 0.25
	degradationRed             = 0.5
	mortalityAmber             = bandDangerMortalityRate // 0.004, shared with the danger tier
	mortalityRed               = 0.008
	shelterAmber               = 0.3
)

type TargetSource uint8

const (
	TargetNone TargetSource = iota
	TargetCursor
	TargetQueued
	TargetHover
)

func (source TargetSource) Label() string {
	return [...]string{"hover or click an outlined tile", "cursor", "queued", "hover"}[source]
}

type LiveabilityTier uint8

const (
	TierNormal LiveabilityTier = iota
	TierAmber
	TierRed
)

// TileLiveability is a presentation-safe reading of one explored land tile.
type TileLiveability struct {
	Available         bool
	Status            string
	Biome             string
	Region            string
	FoodStock         float64
	FoodCap           float64
	WaterStock        float64
	WaterCap          float64
	EcologicalK       float64
	BaselineK         float64
	Degradation       float64
	SeasonalRisk      float64
	ChronicRisk       float64
	CrowdingDecline   float64
	HasRisk           bool
	NaturalShelter    float64
	MovementCost      float64
	ArchaicBands      int
	ArchaicPopulation uint64
	RequiresPassage   bool
	Passage           gameapi.PassageID
	Reachable         bool
}

// CurrentTileLiveability reads the band's own tile with its projected rates.
func CurrentTileLiveability(frame *gameapi.Frame, band *gameapi.Band) TileLiveability {
	if band == nil {
		return TileLiveability{Status: "No active band"}
	}
	summary := summarizeTile(frame, band.TileID)
	if summary.Available {
		summary.Status = "current"
		summary.SeasonalRisk, summary.ChronicRisk, summary.HasRisk = band.SeasonalMortalityRate, band.ChronicMortalityRate, true
	}
	return summary
}

// TargetTile applies the spec's precedence: an arrow-key cursor for this band,
// then an already queued migration, then pointer hover.
func TargetTile(band *gameapi.Band, preview render.MigrationPreview, hover render.TileHover) (gameapi.TileID, TargetSource) {
	if band == nil {
		return 0, TargetNone
	}
	switch {
	case preview.Visible && preview.BandID == band.ID:
		return preview.TileID, TargetCursor
	case band.HasQueuedMigration:
		return band.QueuedMigration, TargetQueued
	case hover.Visible:
		return hover.TileID, TargetHover
	default:
		return 0, TargetNone
	}
}

// TargetTileLiveability reads a candidate tile, taking route-dependent rates
// from the authoritative MigrationCandidates entry when the tile is reachable.
func TargetTileLiveability(frame *gameapi.Frame, band *gameapi.Band, tile gameapi.TileID) TileLiveability {
	summary := summarizeTile(frame, tile)
	if !summary.Available || band == nil {
		return summary
	}
	for _, candidate := range band.MigrationCandidates {
		if candidate.TileID != tile {
			continue
		}
		summary.Reachable = true
		summary.Status = "reachable"
		summary.RequiresPassage, summary.Passage = candidate.RequiresPassage, candidate.Passage
		if candidate.RequiresPassage {
			summary.Status += " via " + candidate.Passage.String()
		}
		summary.SeasonalRisk, summary.ChronicRisk, summary.HasRisk = candidate.SeasonalMortalityRate, candidate.ChronicMortalityRate, true
		summary.CrowdingDecline = candidate.CrowdingDecline
		return summary
	}
	if gameapi.EscarpmentBlocks(frame, band.TileID, tile) {
		summary.Status = "escarpment blocks approach"
	} else {
		summary.Status = "not reachable"
	}
	return summary
}

func summarizeTile(frame *gameapi.Frame, tileID gameapi.TileID) TileLiveability {
	summary := TileLiveability{Status: "Invalid tile"}
	if frame == nil || int(tileID) >= len(frame.Tiles) {
		return summary
	}
	tile := frame.Tiles[tileID]
	switch {
	case !tile.Explored:
		summary.Status = "Unexplored · details hidden"
		return summary
	case !tile.Land:
		summary.Status = "Open water · cannot occupy"
		return summary
	case tile.BaselineK <= 0:
		summary.Status = "Uninhabitable in this climate"
		return summary
	}
	summary.Available = true
	summary.Biome, summary.Region = tile.Biome.String(), tile.Region.String()
	summary.FoodStock, summary.FoodCap = tile.FloraStock+tile.FaunaStock, tile.FloraCap+tile.FaunaCap
	summary.WaterStock, summary.WaterCap = tile.WaterStock, tile.WaterCap
	summary.EcologicalK, summary.BaselineK, summary.Degradation = tile.EcologicalK, tile.BaselineK, tile.Degradation
	summary.NaturalShelter, summary.MovementCost = tile.NaturalShelter, tile.MovementCost
	for _, band := range frame.Bands {
		if band.Species == gameapi.ArchaicHominin && band.TileID == tileID && band.Population > 0 {
			summary.ArchaicBands++
			summary.ArchaicPopulation += uint64(band.Population)
		}
	}
	return summary
}

// LiveabilityRow is one HERE / TARGET comparison line for the Move row.
type LiveabilityRow struct {
	Label      string
	Here       string
	Target     string
	HereTier   LiveabilityTier
	TargetTier LiveabilityTier
	Delta      int // +1 target better, -1 worse, 0 same or unknown
}

// LiveabilityRows builds the eight comparison rows. Absolute tiers color each
// side; Delta compares target to here where both are available.
func LiveabilityRows(band *gameapi.Band, here, target TileLiveability) []LiveabilityRow {
	required := 0.0
	if band != nil && band.LastFoodReport.Turn > 0 {
		required = band.LastFoodReport.RequiredFU
	}
	both := here.Available && target.Available
	row := func(label string, value func(TileLiveability) string, tier func(TileLiveability) LiveabilityTier, higherIsBetter bool, metric func(TileLiveability) float64) LiveabilityRow {
		result := LiveabilityRow{Label: label, Here: "—", Target: "—"}
		if here.Available {
			result.Here, result.HereTier = value(here), tier(here)
		}
		if target.Available {
			result.Target, result.TargetTier = value(target), tier(target)
		}
		if both && metric != nil {
			switch h, t := metric(here), metric(target); {
			case t > h && higherIsBetter, t < h && !higherIsBetter:
				result.Delta = 1
			case t != h:
				result.Delta = -1
			}
		}
		return result
	}
	normal := func(TileLiveability) LiveabilityTier { return TierNormal }
	foodTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case required > 0 && s.FoodStock < required:
			return TierRed
		case required > 0 && s.FoodStock < foodAmberRequirementFactor*required:
			return TierAmber
		}
		return TierNormal
	}
	waterTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case s.WaterCap > 0 && s.WaterStock < waterRedCapFraction*s.WaterCap:
			return TierRed
		case s.WaterCap > 0 && s.WaterStock < waterAmberCapFraction*s.WaterCap:
			return TierAmber
		}
		return TierNormal
	}
	capacityTier := func(s TileLiveability) LiveabilityTier {
		switch {
		case s.Degradation >= degradationRed:
			return TierRed
		case s.Degradation >= degradationAmber:
			return TierAmber
		}
		return TierNormal
	}
	mortalityTier := func(s TileLiveability) LiveabilityTier {
		total := s.SeasonalRisk + s.ChronicRisk
		switch {
		case !s.HasRisk:
			return TierNormal
		case total >= mortalityRed:
			return TierRed
		case total >= mortalityAmber:
			return TierAmber
		}
		return TierNormal
	}
	shelterTier := func(s TileLiveability) LiveabilityTier {
		if s.NaturalShelter < shelterAmber {
			return TierAmber
		}
		return TierNormal
	}
	archaicTier := func(s TileLiveability) LiveabilityTier {
		if s.ArchaicBands > 0 {
			return TierAmber
		}
		return TierNormal
	}
	mortalityValue := func(s TileLiveability) string {
		switch {
		case s.CrowdingDecline > 0:
			return fmt.Sprintf("crowding −%.0f · %.2f%%", s.CrowdingDecline, (s.SeasonalRisk+s.ChronicRisk)*100)
		case s.HasRisk:
			return fmt.Sprintf("%.2f%% · chr %.2f%%", s.SeasonalRisk*100, s.ChronicRisk*100)
		default:
			return "unavailable"
		}
	}
	routeValue := func(s TileLiveability) string {
		if !s.Reachable {
			return "—"
		}
		if s.RequiresPassage {
			return "via " + s.Passage.String()
		}
		return fmt.Sprintf("×%.2f · 1 turn", s.MovementCost)
	}
	archaicValue := func(s TileLiveability) string {
		if s.ArchaicBands == 0 {
			return "none"
		}
		noun := "band"
		if s.ArchaicBands != 1 {
			noun = "bands"
		}
		return fmt.Sprintf("%d %s · pop %d", s.ArchaicBands, noun, s.ArchaicPopulation)
	}
	return []LiveabilityRow{
		row("Biome", func(s TileLiveability) string { return s.Biome }, normal, true, nil),
		row("Food", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.FoodStock, s.FoodCap) }, foodTier, true, func(s TileLiveability) float64 { return s.FoodStock }),
		row("Capacity", func(s TileLiveability) string {
			return fmt.Sprintf("%.0f · %.0f%% degr.", s.EcologicalK, s.Degradation*100)
		}, capacityTier, true, func(s TileLiveability) float64 { return s.EcologicalK }),
		row("Water", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.WaterStock, s.WaterCap) }, waterTier, true, func(s TileLiveability) float64 { return s.WaterStock }),
		row("Shelter", func(s TileLiveability) string { return fmt.Sprintf("%.0f%%", s.NaturalShelter*100) }, shelterTier, true, func(s TileLiveability) float64 { return s.NaturalShelter }),
		row("Mortality", mortalityValue, mortalityTier, false, func(s TileLiveability) float64 { return s.SeasonalRisk + s.ChronicRisk + s.CrowdingDecline }),
		row("Route", routeValue, normal, true, nil),
		row("Archaic", archaicValue, archaicTier, true, nil),
	}
}
