package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/adsouza/africa2ice/internal/domain"
)

// Version 2 stores event facts; version 1 summaries remain readable.
const SaveSchemaVersion = 2

// OldestSupportedSaveSchemaVersion is the lowest schema RestoreWorld accepts.
// internal/adapters/storage/testdata/oldest_supported_save_v1.json.gz is frozen
// at this version; raising it is a deliberate compatibility break, not a
// consequence of bumping SaveSchemaVersion.
const OldestSupportedSaveSchemaVersion = 1

type AlgorithmVersions struct {
	GeographyAlgorithm           string `json:"geography_algorithm"`
	NaturalShelterMaskAlgorithm  string `json:"natural_shelter_mask_algorithm"`
	CampaignClockAlgorithm       string `json:"campaign_clock_algorithm"`
	ClimateAlgorithm             string `json:"climate_algorithm"`
	TemperatureAlgorithm         string `json:"temperature_algorithm"`
	MacroEventAlgorithm          string `json:"macro_event_algorithm"`
	ExplorationAlgorithm         string `json:"exploration_algorithm"`
	BandAlgorithm                string `json:"band_algorithm"`
	ArchaicPolicyAlgorithm       string `json:"archaic_policy_algorithm"`
	AssignmentAlgorithm          string `json:"assignment_algorithm"`
	FoodStorageAlgorithm         string `json:"food_storage_algorithm"`
	FoodConversionAlgorithm      string `json:"food_conversion_algorithm"`
	ForagingAlgorithm            string `json:"foraging_algorithm"`
	HuntingAlgorithm             string `json:"hunting_algorithm"`
	MegafaunaAlgorithm           string `json:"megafauna_algorithm"`
	HuntingRiskAlgorithm         string `json:"hunting_risk_algorithm"`
	FaunaProfileAlgorithm        string `json:"fauna_profile_algorithm"`
	ShelterAlgorithm             string `json:"shelter_algorithm"`
	TechnologyOwnershipAlgorithm string `json:"technology_ownership_algorithm"`
	ResearchProductionAlgorithm  string `json:"research_production_algorithm"`
	KnowledgeContactAlgorithm    string `json:"knowledge_contact_algorithm"`
	KnowledgeDiffusionAlgorithm  string `json:"knowledge_diffusion_algorithm"`
	HeritableStateAlgorithm      string `json:"heritable_state_algorithm"`
	GeneticSelectionAlgorithm    string `json:"genetic_selection_algorithm"`
	GeneFlowAlgorithm            string `json:"gene_flow_algorithm"`
	MutationAlgorithm            string `json:"mutation_algorithm"`
	MovementAlgorithm            string `json:"movement_algorithm"`
	MovementCostAlgorithm        string `json:"movement_cost_algorithm"`
	PassageAlgorithm             string `json:"passage_algorithm"`
	ResourceAlgorithm            string `json:"resource_algorithm"`
	HazardAlgorithm              string `json:"hazard_algorithm"`
	KinSupportAlgorithm          string `json:"kin_support_algorithm"`
	PopulationRoundingAlgorithm  string `json:"population_rounding_algorithm"`
	RNGAlgorithm                 string `json:"rng_algorithm"`
}

var supportedAlgorithms = AlgorithmVersions{
	GeographyAlgorithm: "dispersal-map-v5", NaturalShelterMaskAlgorithm: "authored-ellipse-v1",
	CampaignClockAlgorithm: "four-era-v1", ClimateAlgorithm: "hybrid-abrupt-moisture-v1",
	TemperatureAlgorithm: "lat-elev-offset-v1", MacroEventAlgorithm: "bounded-regional-v1",
	ExplorationAlgorithm: "sapiens-frontier-v1", BandAlgorithm: "capacity-safe-half-global-cap-v2",
	ArchaicPolicyAlgorithm: "ranked-pressure-v1", AssignmentAlgorithm: "proportional-basis-points-v1",
	FoodStorageAlgorithm: "population-food-turns-v1", FoodConversionAlgorithm: "normalized-source-v1",
	ForagingAlgorithm: "linear-shared-flora-v1", HuntingAlgorithm: "linear-shared-fauna-v1",
	MegafaunaAlgorithm: "linear-megafauna-v1", HuntingRiskAlgorithm: "linear-share-work-risk-v1",
	FaunaProfileAlgorithm: "region-biome-v1", ShelterAlgorithm: "saturating-share-terrain-v1",
	TechnologyOwnershipAlgorithm: "band-local-v1", ResearchProductionAlgorithm: "saturating-toolcraft-v1",
	KnowledgeContactAlgorithm: "co-located-cross-species-v1", KnowledgeDiffusionAlgorithm: "stacked-acquired-v1",
	HeritableStateAlgorithm: "band-six-trait-v1", GeneticSelectionAlgorithm: "trait-functions-v1",
	GeneFlowAlgorithm: "local-reciprocal-v1", MutationAlgorithm: "rare-emergence-v1",
	MovementAlgorithm: "eight-way-escarpment-corners-v2", MovementCostAlgorithm: "destination-vegetation-v1",
	PassageAlgorithm: "named-asymmetric-v1", ResourceAlgorithm: "toward-cap-v1",
	HazardAlgorithm: "split-v1", KinSupportAlgorithm: "saturating-kin-acute-v1", PopulationRoundingAlgorithm: "stochastic-v1",
	RNGAlgorithm: "pcg-splitmix-v1",
}

// SupportedCompatibility returns the persisted compatibility identity release
// tooling reports: the schema this build writes, the oldest schema it still
// reads, and the algorithm versions it requires. RestoreWorld enforces the
// schema range itself; this reports it so a release record answers "can this
// build open my save?". Keeping this a value prevents tooling from mutating
// the runtime contract.
func SupportedCompatibility() (int, int, AlgorithmVersions) {
	return SaveSchemaVersion, OldestSupportedSaveSchemaVersion, supportedAlgorithms
}

type SaveState struct {
	EasyMode      bool `json:"easy_mode,omitempty"`
	SchemaVersion int  `json:"schema_version"`
	AlgorithmVersions
	WorldRevision      uint64                              `json:"world_revision"`
	WorldSeed          uint64                              `json:"world_seed"`
	GridWidth          int                                 `json:"grid_width"`
	GridHeight         int                                 `json:"grid_height"`
	Turn               int                                 `json:"turn"`
	Result             uint8                               `json:"result"`
	NextBandID         uint64                              `json:"next_band_id"`
	Bands              []BandSave                          `json:"bands"`
	Events             []EventSave                         `json:"events,omitempty"`
	Tiles              [domain.TileCount]TileSave          `json:"tiles"`
	ExploredTiles      [domain.ExplorationWordCount]uint64 `json:"explored_tiles"`
	EstablishedRegions []uint8                             `json:"sapiens_established_regions"`
	RNGState           []byte                              `json:"rng_state"`
}

type TileSave struct {
	Degradation float64 `json:"degradation"`
	Flora       float64 `json:"flora"`
	Fauna       float64 `json:"fauna"`
	Water       float64 `json:"water"`
}

type BandSave struct {
	ID                  uint64                              `json:"id"`
	Species             uint8                               `json:"species"`
	TileID              uint16                              `json:"tile_id"`
	Population          uint32                              `json:"population"`
	Health              float64                             `json:"health"`
	StoredFood          float64                             `json:"stored_food"`
	Allocation          [domain.AssignmentCount]uint16      `json:"allocation_bp"`
	AcquiredTech        uint16                              `json:"acquired_tech"`
	ResearchTarget      uint8                               `json:"research_target"`
	HasResearchTarget   bool                                `json:"has_research_target"`
	ResearchProgress    [domain.TechCount]float64           `json:"research_progress"`
	Heritable           [domain.HeritableTraitCount]float64 `json:"heritable_state"`
	LastFoodReport      FoodReportSave                      `json:"last_food_report"`
	LastMortality       MortalitySave                       `json:"last_mortality"`
	LastOutcomeReport   OutcomeReportSave                   `json:"last_outcome_report"`
	SpatialActionUsed   bool                                `json:"spatial_action_used"`
	QueuedMigration     uint16                              `json:"queued_migration"`
	QueuedOrigin        uint16                              `json:"queued_origin"`
	QueuedPassage       uint8                               `json:"queued_passage"`
	QueueUsesPassage    bool                                `json:"queue_uses_passage"`
	HasQueuedMigration  bool                                `json:"has_queued_migration"`
	InterbreedTarget    uint64                              `json:"interbreed_target"`
	HasInterbreedTarget bool                                `json:"has_interbreed_target"`
}

type FoodReportSave struct {
	Turn       int     `json:"turn"`
	RequiredFU float64 `json:"required_fu"`
	DeficitFU  float64 `json:"deficit_fu"`
}
type MortalitySave struct {
	Starvation float64 `json:"starvation"`
	Seasonal   float64 `json:"seasonal"`
	Chronic    float64 `json:"chronic"`
	Macro      float64 `json:"macro"`
	Acute      float64 `json:"acute"`
}

type OutcomeReportSave struct {
	Turn                    int     `json:"turn"`
	StartingPopulation      uint32  `json:"starting_population"`
	EndingPopulation        uint32  `json:"ending_population"`
	Growth                  float64 `json:"growth"`
	StartingHealth          float64 `json:"starting_health"`
	EndingHealth            float64 `json:"ending_health"`
	NutritionDelta          float64 `json:"nutrition_delta"`
	WaterHealthLoss         float64 `json:"water_health_loss"`
	DiseaseHealthLoss       float64 `json:"disease_health_loss"`
	GeneticBurdenHealthLoss float64 `json:"genetic_burden_health_loss"`
	MacroHealthLoss         float64 `json:"macro_health_loss"`
	AcuteDiseaseHealthLoss  float64 `json:"acute_disease_health_loss"`
}

type EventSave struct {
	Turn    int               `json:"turn"`
	Kind    uint8             `json:"kind"`
	BandID  uint64            `json:"band_id"`
	TileID  uint16            `json:"tile_id"`
	Region  uint8             `json:"region"`
	Summary string            `json:"summary,omitempty"`
	Details *EventDetailsSave `json:"details,omitempty"`
}

// EventDetailsSave contains no presentation text. A nil value identifies an
// imported legacy event whose original summary must be retained.
type EventDetailsSave struct {
	ParentBandID   uint64 `json:"parent_band_id,omitempty"`
	Technology     uint8  `json:"technology,omitempty"`
	AcuteKind      uint8  `json:"acute_kind,omitempty"`
	Species        uint8  `json:"species,omitempty"`
	MortalityCause uint8  `json:"mortality_cause,omitempty"`
}

// SavedHomoSapiens and SavedArchaicHominin name the BandSave.Species encoding.
// The wire form is a bare uint8, so without these an adapter has to hardcode
// the domain enum's ordering to read a save it is not allowed to import.
const (
	SavedHomoSapiens    uint8 = uint8(domain.HomoSapiens)
	SavedArchaicHominin uint8 = uint8(domain.ArchaicHominin)
)

// PopulationBySpecies totals the saved bands. Repository adapters record these
// in slot metadata, so the classification lives here with the encoding it
// depends on rather than being retyped once per adapter.
func (save SaveState) PopulationBySpecies() (sapiens, archaic uint64) {
	for _, band := range save.Bands {
		switch band.Species {
		case SavedHomoSapiens:
			sapiens += uint64(band.Population)
		case SavedArchaicHominin:
			archaic += uint64(band.Population)
		}
	}
	return sapiens, archaic
}

func SaveStateFromWorld(world *domain.World, revision uint64) (SaveState, error) {
	state, err := world.ExportState()
	if err != nil {
		return SaveState{}, err
	}
	save := SaveState{
		EasyMode: state.EasyMode, SchemaVersion: SaveSchemaVersion, AlgorithmVersions: supportedAlgorithms, WorldRevision: revision,
		WorldSeed: state.Seed, GridWidth: domain.MapWidth, GridHeight: domain.MapHeight, Turn: state.Turn,
		Result: uint8(state.Result), NextBandID: uint64(state.NextBandID), ExploredTiles: state.ExploredTiles,
		RNGState: append([]byte(nil), state.RNGState...),
	}
	for region := domain.Region(0); region < domain.RegionCount; region++ {
		if state.EstablishedRegions&(1<<region) != 0 {
			save.EstablishedRegions = append(save.EstablishedRegions, uint8(region))
		}
	}
	for id, tile := range state.Tiles {
		save.Tiles[id] = TileSave{tile.Degradation, tile.Stock.Flora, tile.Stock.Fauna, tile.Stock.Water}
	}
	for _, band := range state.Bands {
		item := BandSave{
			ID: uint64(band.ID), Species: uint8(band.Species), TileID: uint16(band.TileID), Population: uint32(band.Population), Health: float64(band.Health), StoredFood: float64(band.StoredFood),
			AcquiredTech: band.Technology.Acquired, ResearchTarget: uint8(band.Technology.Target), HasResearchTarget: band.Technology.HasTarget,
			LastFoodReport: FoodReportSave{band.LastFoodReport.Turn, band.LastFoodReport.RequiredFU, band.LastFoodReport.DeficitFU},
			LastMortality:  MortalitySave{band.LastMortality.Starvation, band.LastMortality.Seasonal, band.LastMortality.Chronic, band.LastMortality.Macro, band.LastMortality.Acute},
			LastOutcomeReport: OutcomeReportSave{
				Turn: band.LastOutcomeReport.Turn, StartingPopulation: uint32(band.LastOutcomeReport.StartingPopulation), EndingPopulation: uint32(band.LastOutcomeReport.EndingPopulation), Growth: band.LastOutcomeReport.Growth,
				StartingHealth: float64(band.LastOutcomeReport.StartingHealth), EndingHealth: float64(band.LastOutcomeReport.EndingHealth), NutritionDelta: band.LastOutcomeReport.NutritionDelta,
				WaterHealthLoss: band.LastOutcomeReport.WaterHealthLoss, DiseaseHealthLoss: band.LastOutcomeReport.DiseaseHealthLoss, GeneticBurdenHealthLoss: band.LastOutcomeReport.GeneticBurdenHealthLoss,
				MacroHealthLoss: band.LastOutcomeReport.MacroHealthLoss, AcuteDiseaseHealthLoss: band.LastOutcomeReport.AcuteDiseaseHealthLoss,
			},
			SpatialActionUsed: band.SpatialActionUsed, QueuedMigration: uint16(band.QueuedMigration), QueuedOrigin: uint16(band.QueuedOrigin), QueuedPassage: uint8(band.QueuedPassage), QueueUsesPassage: band.QueueUsesPassage, HasQueuedMigration: band.HasQueuedMigration,
			InterbreedTarget: uint64(band.InterbreedTarget), HasInterbreedTarget: band.HasInterbreedTarget,
		}
		for index, value := range band.Allocation {
			item.Allocation[index] = uint16(value)
		}
		item.ResearchProgress = band.Technology.Progress
		for index, value := range band.Heritable {
			item.Heritable[index] = float64(value)
		}
		save.Bands = append(save.Bands, item)
	}
	for _, event := range state.Events {
		item := EventSave{Turn: event.Turn, Kind: uint8(event.Kind), BandID: uint64(event.BandID), TileID: uint16(event.TileID), Region: uint8(event.Region), Summary: event.LegacySummary}
		if event.LegacySummary == "" {
			item.Details = &EventDetailsSave{ParentBandID: uint64(event.Details.ParentBandID), Technology: uint8(event.Details.Technology), AcuteKind: uint8(event.Details.AcuteKind), Species: uint8(event.Details.Species), MortalityCause: uint8(event.Details.MortalityCause)}
		}
		save.Events = append(save.Events, item)
	}
	return save, nil
}

func (save SaveState) RestoreWorld() (*domain.World, error) {
	if save.SchemaVersion < OldestSupportedSaveSchemaVersion || save.SchemaVersion > SaveSchemaVersion {
		return nil, fmt.Errorf("unsupported save schema %d", save.SchemaVersion)
	}
	if save.AlgorithmVersions != supportedAlgorithms {
		return nil, fmt.Errorf("unsupported algorithm version")
	}
	if save.GridWidth != domain.MapWidth || save.GridHeight != domain.MapHeight {
		return nil, fmt.Errorf("invalid grid dimensions")
	}
	state := domain.State{EasyMode: save.EasyMode, Seed: save.WorldSeed, Turn: save.Turn, Result: domain.CampaignResult(save.Result), NextBandID: domain.BandID(save.NextBandID), ExploredTiles: save.ExploredTiles, RNGState: append([]byte(nil), save.RNGState...)}
	previous := -1
	for _, region := range save.EstablishedRegions {
		if int(region) <= previous || region >= uint8(domain.RegionCount) {
			return nil, fmt.Errorf("invalid established regions")
		}
		previous = int(region)
		state.EstablishedRegions |= 1 << region
	}
	for id, tile := range save.Tiles {
		if !finite(tile.Degradation) || !finite(tile.Flora) || !finite(tile.Fauna) || !finite(tile.Water) {
			return nil, fmt.Errorf("non-finite tile %d", id)
		}
		state.Tiles[id] = domain.TileState{Degradation: tile.Degradation, Stock: domain.ResourceVector{Flora: tile.Flora, Fauna: tile.Fauna, Water: tile.Water}}
	}
	for _, item := range save.Bands {
		band := domain.Band{
			ID: domain.BandID(item.ID), Species: domain.Species(item.Species), TileID: domain.TileID(item.TileID), Population: domain.Population(item.Population), Health: domain.Health(item.Health), StoredFood: domain.FU(item.StoredFood),
			Technology:     domain.TechnologyState{Acquired: item.AcquiredTech, Target: domain.Technology(item.ResearchTarget), HasTarget: item.HasResearchTarget},
			LastFoodReport: domain.FoodTurnReport{Turn: item.LastFoodReport.Turn, RequiredFU: item.LastFoodReport.RequiredFU, DeficitFU: item.LastFoodReport.DeficitFU},
			LastMortality:  domain.MortalityReport{Starvation: item.LastMortality.Starvation, Seasonal: item.LastMortality.Seasonal, Chronic: item.LastMortality.Chronic, Macro: item.LastMortality.Macro, Acute: item.LastMortality.Acute},
			LastOutcomeReport: domain.OutcomeReport{
				Turn: item.LastOutcomeReport.Turn, StartingPopulation: domain.Population(item.LastOutcomeReport.StartingPopulation), EndingPopulation: domain.Population(item.LastOutcomeReport.EndingPopulation), Growth: item.LastOutcomeReport.Growth,
				StartingHealth: domain.Health(item.LastOutcomeReport.StartingHealth), EndingHealth: domain.Health(item.LastOutcomeReport.EndingHealth), NutritionDelta: item.LastOutcomeReport.NutritionDelta,
				WaterHealthLoss: item.LastOutcomeReport.WaterHealthLoss, DiseaseHealthLoss: item.LastOutcomeReport.DiseaseHealthLoss, GeneticBurdenHealthLoss: item.LastOutcomeReport.GeneticBurdenHealthLoss,
				MacroHealthLoss: item.LastOutcomeReport.MacroHealthLoss, AcuteDiseaseHealthLoss: item.LastOutcomeReport.AcuteDiseaseHealthLoss,
			},
			SpatialActionUsed: item.SpatialActionUsed, QueuedMigration: domain.TileID(item.QueuedMigration), QueuedOrigin: domain.TileID(item.QueuedOrigin), QueuedPassage: domain.PassageID(item.QueuedPassage), QueueUsesPassage: item.QueueUsesPassage, HasQueuedMigration: item.HasQueuedMigration,
			InterbreedTarget: domain.BandID(item.InterbreedTarget), HasInterbreedTarget: item.HasInterbreedTarget,
		}
		for index, value := range item.Allocation {
			band.Allocation[index] = domain.AssignmentBP(value)
		}
		band.Technology.Progress = item.ResearchProgress
		for index, value := range item.Heritable {
			band.Heritable[index] = domain.TraitValue(value)
		}
		state.Bands = append(state.Bands, band)
	}
	for _, item := range save.Events {
		event := domain.Event{Turn: item.Turn, Kind: domain.EventKind(item.Kind), BandID: domain.BandID(item.BandID), TileID: domain.TileID(item.TileID), Region: domain.Region(item.Region), LegacySummary: item.Summary}
		if item.Details == nil {
			if item.Summary == "" {
				return nil, fmt.Errorf("event has neither facts nor legacy summary")
			}
		} else {
			if save.SchemaVersion == 1 || item.Summary != "" {
				return nil, fmt.Errorf("event facts conflict with legacy encoding")
			}
			d := item.Details
			event.Details = domain.EventDetails{ParentBandID: domain.BandID(d.ParentBandID), Technology: domain.Technology(d.Technology), AcuteKind: domain.AcuteKind(d.AcuteKind), Species: domain.Species(d.Species), MortalityCause: domain.MortalityCause(d.MortalityCause)}
		}
		state.Events = append(state.Events, event)
	}
	return domain.RestoreWorld(state)
}

func EncodeSaveState(save SaveState) ([]byte, error) { return json.Marshal(save) }

func DecodeSaveState(data []byte) (SaveState, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var save SaveState
	if err := decoder.Decode(&save); err != nil {
		return SaveState{}, err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return SaveState{}, err
	}
	return save, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 }

func (service *GameService) ExportSaveState() (SaveState, error) {
	return SaveStateFromWorld(service.world, service.worldRevision)
}

// StateHash returns the canonical campaign-state hash. The normalized save
// payload is the application-owned durable representation, so repository
// metadata, timestamps, operation IDs, and presentation state cannot affect
// this value.
func (service *GameService) StateHash() (string, error) {
	state, err := service.ExportSaveState()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum), nil
}
