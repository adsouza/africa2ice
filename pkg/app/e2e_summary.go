package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// E2ESummary is the bounded semantic browser-test view of the already
// published frame. It contains no command surface and requests no snapshot.
type E2ESummary struct {
	CalendarProgress  float64          `json:"calendar_progress"`
	Era               string           `json:"era"`
	ExploredHash      string           `json:"explored_hash"`
	ExploredTileIDs   []gameapi.TileID `json:"explored_tile_ids"`
	FirstSapiensBand  E2EBandSummary   `json:"first_sapiens_band"`
	SapiensPopulation uint64           `json:"sapiens_population"`
	Turn              int              `json:"turn"`
	WorldRevision     uint64           `json:"world_revision"`
	YearBP            int              `json:"year_bp"`
}

type E2EBandSummary struct {
	Health      float64          `json:"health"`
	ID          gameapi.BandID   `json:"id"`
	LastFood    E2EFoodReport    `json:"last_food_report"`
	LastOutcome E2EOutcomeReport `json:"last_outcome_report"`
	Population  uint32           `json:"population"`
	Present     bool             `json:"present"`
	StoredFood  float64          `json:"stored_food"`
	TileID      gameapi.TileID   `json:"tile_id"`
}

type E2EFoodReport struct {
	DeficitFU  float64 `json:"deficit_fu"`
	RequiredFU float64 `json:"required_fu"`
	Turn       int     `json:"turn"`
}

type E2EOutcomeReport struct {
	AcuteDiseaseHealthLoss  float64 `json:"acute_disease_health_loss"`
	DiseaseHealthLoss       float64 `json:"disease_health_loss"`
	EndingHealth            float64 `json:"ending_health"`
	EndingPopulation        uint32  `json:"ending_population"`
	GeneticBurdenHealthLoss float64 `json:"genetic_burden_health_loss"`
	Growth                  float64 `json:"growth"`
	MacroHealthLoss         float64 `json:"macro_health_loss"`
	NutritionDelta          float64 `json:"nutrition_delta"`
	StartingHealth          float64 `json:"starting_health"`
	StartingPopulation      uint32  `json:"starting_population"`
	Turn                    int     `json:"turn"`
	WaterHealthLoss         float64 `json:"water_health_loss"`
}

func e2eSummaryJSON(frame *gameapi.Frame) string {
	if frame == nil {
		return "null"
	}
	summary := E2ESummary{
		CalendarProgress: frame.CalendarProgress, Era: frame.Era.String(),
		ExploredTileIDs: make([]gameapi.TileID, 0), Turn: frame.Turn,
		WorldRevision: frame.WorldRevision, YearBP: frame.YearBP,
	}
	explored := make([]byte, len(frame.Tiles))
	for _, tile := range frame.Tiles {
		if !tile.Explored || int(tile.ID) >= len(explored) {
			continue
		}
		explored[tile.ID] = 1
		summary.ExploredTileIDs = append(summary.ExploredTileIDs, tile.ID)
	}
	hash := sha256.Sum256(explored)
	summary.ExploredHash = hex.EncodeToString(hash[:])
	var first *gameapi.Band
	for index := range frame.Bands {
		band := &frame.Bands[index]
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		summary.SapiensPopulation += uint64(band.Population)
		if first == nil || band.ID < first.ID {
			first = band
		}
	}
	if first != nil {
		summary.FirstSapiensBand = E2EBandSummary{
			Health: first.Health, ID: first.ID, Population: first.Population, Present: true,
			StoredFood: first.StoredFood, TileID: first.TileID,
			LastFood: E2EFoodReport{DeficitFU: first.LastFoodReport.DeficitFU, RequiredFU: first.LastFoodReport.RequiredFU, Turn: first.LastFoodReport.Turn},
			LastOutcome: E2EOutcomeReport{
				AcuteDiseaseHealthLoss: first.LastOutcomeReport.AcuteDiseaseHealthLoss,
				DiseaseHealthLoss:      first.LastOutcomeReport.DiseaseHealthLoss, EndingHealth: first.LastOutcomeReport.EndingHealth,
				EndingPopulation: first.LastOutcomeReport.EndingPopulation, GeneticBurdenHealthLoss: first.LastOutcomeReport.GeneticBurdenHealthLoss,
				Growth: first.LastOutcomeReport.Growth, MacroHealthLoss: first.LastOutcomeReport.MacroHealthLoss,
				NutritionDelta: first.LastOutcomeReport.NutritionDelta, StartingHealth: first.LastOutcomeReport.StartingHealth,
				StartingPopulation: first.LastOutcomeReport.StartingPopulation, Turn: first.LastOutcomeReport.Turn,
				WaterHealthLoss: first.LastOutcomeReport.WaterHealthLoss,
			},
		}
	}
	payload, err := json.Marshal(summary)
	if err != nil {
		return "null"
	}
	return string(payload)
}
