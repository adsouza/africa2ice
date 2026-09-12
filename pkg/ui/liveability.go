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
	// Capacity is tiered by how much of the tile would be in use once the band
	// arrives, not by degradation alone: a smaller capacity the band fits
	// several times over is not a warning (user-reported). The amber point is
	// the domain's own crowding threshold — domain.World.BandStress divides a
	// tile's total resident population by its capacity and splits a band past
	// SplitStressThreshold — so the panel and the simulation agree on when a
	// tile is crowded.
	capacityOccupancyAmber = gameapi.SplitStressThreshold // 0.67
	capacityOccupancyRed   = 1.0
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
	Available       bool
	Status          string
	Biome           string
	Region          string
	NearbyLake      string
	FoodStock       float64
	FoodCap         float64
	WaterStock      float64
	WaterCap        float64
	EcologicalK     float64
	BaselineK       float64
	Degradation     float64
	SeasonalRisk    float64
	ChronicRisk     float64
	CrowdingDecline float64
	HasRisk         bool
	NaturalShelter  float64
	MovementCost    float64
	// ResidentBands and ResidentPopulation count every band already on the
	// tile *except* the one being read for, of any species. Excluding it lets
	// one occupancy formula serve both columns: HERE adds the band back to a
	// tile it already occupies and TARGET adds it to a tile it has yet to
	// reach, and both are "who would be here once this band is".
	ResidentBands      int
	ResidentPopulation uint64
	// ArchaicBands is kept alongside ResidentBands so the Others row can name
	// the species while every neighbour is archaic. The matching population is
	// not kept: the row reports ResidentPopulation, and an archaic-only
	// headcount had no reader left.
	ArchaicBands    int
	RequiresPassage bool
	Passage         gameapi.PassageID
	Reachable       bool
}

// CurrentTileLiveability reads the band's own tile with its projected rates.
func CurrentTileLiveability(frame *gameapi.Frame, band *gameapi.Band) TileLiveability {
	if band == nil {
		return TileLiveability{Status: "No active band"}
	}
	summary := summarizeTile(frame, band.TileID, band)
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
	summary := summarizeTile(frame, tile, band)
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

func summarizeTile(frame *gameapi.Frame, tileID gameapi.TileID, self *gameapi.Band) TileLiveability {
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
	summary.NearbyLake = tile.NearbyLake
	summary.FoodStock, summary.FoodCap = tile.FloraStock+tile.FaunaStock, tile.FloraCap+tile.FaunaCap
	summary.WaterStock, summary.WaterCap = tile.WaterStock, tile.WaterCap
	summary.EcologicalK, summary.BaselineK, summary.Degradation = tile.EcologicalK, tile.BaselineK, tile.Degradation
	summary.NaturalShelter, summary.MovementCost = tile.NaturalShelter, tile.MovementCost
	for _, resident := range frame.Bands {
		if resident.TileID != tileID || resident.Population == 0 || (self != nil && resident.ID == self.ID) {
			continue
		}
		summary.ResidentBands++
		summary.ResidentPopulation += uint64(resident.Population)
		if resident.Species == gameapi.ArchaicHominin {
			summary.ArchaicBands++
		}
	}
	return summary
}

// ProjectedOccupancy is the share of the tile's capacity in use once a band of
// arriving people is present, alongside the residents already counted. It is
// the presentation twin of domain.World.BandStress, over EcologicalK rather
// than BaselineK so that the ratio matches the capacity displayed beside it.
func (s TileLiveability) ProjectedOccupancy(arriving uint64) float64 {
	people := float64(s.ResidentPopulation + arriving)
	if s.EcologicalK <= 0 {
		if people > 0 {
			return capacityOccupancyRed // no capacity at all is full by definition
		}
		return 0
	}
	return people / s.EcologicalK
}

// LiveabilityRow is one HERE / TARGET comparison line for the Move row.
type LiveabilityRow struct {
	Label      string
	Here       string
	Target     string
	HereTier   LiveabilityTier
	TargetTier LiveabilityTier
	Delta      int // +1 target better, -1 worse, 0 same or unknown
	// DeltaMaterial reports whether the difference Delta describes has
	// consequences: a mark is only worth a warning colour when at least one
	// side is already out of TierNormal. Two comfortable tiles can differ by a
	// lot without the band feeling any of it (user-reported).
	DeltaMaterial bool
}

// LiveabilityRows builds the ten comparison rows. Absolute tiers color each
// side; Delta compares target to here where both are available.
func LiveabilityRows(band *gameapi.Band, here, target TileLiveability) []LiveabilityRow {
	required := 0.0
	if band != nil && band.LastFoodReport.Turn > 0 {
		required = band.LastFoodReport.RequiredFU
	}
	population := uint64(0)
	if band != nil {
		population = uint64(band.Population)
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
		// The mark is a claim about the two numbers on screen, so it compares
		// the formatted values: metrics are raw float64 behind a rounded
		// formatter, and 211.6 next to 212.4 both render "212" (user-reported).
		if both && metric != nil && result.Here != result.Target {
			switch h, t := metric(here), metric(target); {
			case t > h && higherIsBetter, t < h && !higherIsBetter:
				result.Delta = 1
			case t != h:
				result.Delta = -1
			}
			// A difference only earns a warning colour when at least one side
			// is already out of TierNormal; between two comfortable tiles it is
			// real but inconsequential, and colouring it cried wolf.
			result.DeltaMaterial = result.Delta != 0 && (result.HereTier != TierNormal || result.TargetTier != TierNormal)
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
	occupancy := func(s TileLiveability) float64 { return s.ProjectedOccupancy(population) }
	capacityTier := func(s TileLiveability) LiveabilityTier {
		switch used := occupancy(s); {
		case s.Degradation >= degradationRed, used >= capacityOccupancyRed:
			return TierRed
		case s.Degradation >= degradationAmber, used >= capacityOccupancyAmber:
			return TierAmber
		}
		return TierNormal
	}
	// Degradation shows as the gap between present and baseline capacity, in
	// the same "current / potential" idiom the Food and Water rows use, and
	// only once there is a gap: "0% degr." on every undegraded tile spent the
	// column's width saying nothing, and a third percentage alongside the
	// occupancy figure overflowed it outright (TestMoveGridValuesFitTheirColumns).
	capacityValue := func(s TileLiveability) string {
		capacity := fmt.Sprintf("%.0f", s.EcologicalK)
		if s.Degradation > 0 {
			capacity = fmt.Sprintf("%.0f/%.0f", s.EcologicalK, s.BaselineK)
		}
		return fmt.Sprintf("%s · %.0f%% full", capacity, occupancy(s)*100)
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
	mortalityValue := func(s TileLiveability) string {
		switch {
		case s.CrowdingDecline > 0:
			// "crowd" rather than "crowding": the headcount and the rate
			// together overflowed the column once a delta mark was appended,
			// and the shorter word fits even implausible values
			// (TestMoveGridValuesFitTheirColumns).
			return fmt.Sprintf("crowd −%.0f · %.2f%%", s.CrowdingDecline, (s.SeasonalRisk+s.ChronicRisk)*100)
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
	// Every band already on the tile, of any species. This row is purely
	// informational: an archaic neighbour is the precondition for
	// interbreeding, so tiering it amber painted the row's own reason for
	// existing as a hazard (user-reported). A neighbour's competition for the
	// tile shows up on Capacity instead, as occupancy the band would join.
	othersValue := func(s TileLiveability) string {
		if s.ResidentBands == 0 {
			return "none"
		}
		noun := "bands"
		switch {
		case s.ArchaicBands == s.ResidentBands:
			noun = "archaic" // keeps the interbreeding cue the tier used to carry
		case s.ResidentBands == 1:
			noun = "band"
		}
		return fmt.Sprintf("%d %s · pop %d", s.ResidentBands, noun, s.ResidentPopulation)
	}
	return []LiveabilityRow{
		row("Biome", func(s TileLiveability) string { return s.Biome }, normal, true, nil),
		row("Region", func(s TileLiveability) string { return s.Region }, normal, true, nil),
		row("Nearby lake", func(s TileLiveability) string {
			if s.NearbyLake == "" {
				return "—"
			}
			return s.NearbyLake
		}, normal, true, nil),
		row("Food", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.FoodStock, s.FoodCap) }, foodTier, true, func(s TileLiveability) float64 { return s.FoodStock }),
		row("Capacity", capacityValue, capacityTier, false, occupancy),
		row("Water", func(s TileLiveability) string { return fmt.Sprintf("%.0f / %.0f", s.WaterStock, s.WaterCap) }, waterTier, true, func(s TileLiveability) float64 { return s.WaterStock }),
		row("Shelter", func(s TileLiveability) string { return fmt.Sprintf("%.0f%%", s.NaturalShelter*100) }, shelterTier, true, func(s TileLiveability) float64 { return s.NaturalShelter }),
		row("Mortality", mortalityValue, mortalityTier, false, func(s TileLiveability) float64 { return s.SeasonalRisk + s.ChronicRisk }),
		row("Route", routeValue, normal, true, nil),
		row("Others", othersValue, normal, true, nil),
	}
}
