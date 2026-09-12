package gameapi

type TileID uint16
type BandID uint64

type Frame struct {
	EasyMode                  bool
	WorldRevision             uint64
	TerrainRevision           uint64
	Turn                      int
	YearBP                    int
	Era                       CampaignEra
	CalendarProgress          float64
	Season                    Season
	Climate                   ClimateSummary
	MacroEpisodes             []MacroEpisodeSummary
	Passages                  []Passage
	Escarpments               []Escarpment
	SapiensEstablishedRegions []Region
	Tiles                     []Tile
	Bands                     []Band
	CampaignResult            CampaignResult
	Events                    []Event
}

type ClimateSummary struct {
	LongTermTempOffset float64
	SeasonalTempOffset float64
	ClimateNoise       float64
	GlobalTempOffset   float64
	MoistureOffset     float64
	AridityIndex       float64
	Epoch              ClimateEpoch
	RegionalAbrupt     [RegionCount]float64
}

type ClimateEpoch uint8

const (
	HumidOptimum ClimateEpoch = iota
	AridTransition
	GlacialMaximum
	ClimateEpochCount
)

func (v ClimateEpoch) String() string {
	return enumString(int(v), []string{"Humid Optimum", "Arid Transition", "Glacial Maximum"}, "ClimateEpoch")
}

type MacroEpisodeSummary struct {
	Episode MacroEpisode
	Warned  bool
	Current bool
	Elapsed bool
}

type Passage struct {
	ID       PassageID
	From     TileID
	To       TileID
	Status   PassageStatus
	Cost     float64
	Explored bool
}

// Escarpment is an explored, impassable boundary between two cardinally
// adjacent land tiles.
type Escarpment struct {
	Name          string
	First, Second TileID
}

type Tile struct {
	ID                TileID
	X                 int
	Y                 int
	Latitude          float64
	Longitude         float64
	Land              bool
	Region            Region
	NearbyLake        string // Named lake near this land tile; empty when uncataloged.
	WaterBody         string // Geographic name for water tiles; empty on land.
	ElevationKm       float64
	Biome             Biome
	Explored          bool
	NaturalShelter    float64
	LocalTemperatureC float64
	MovementCost      float64
	BaseMoisture      float64
	VegetationIndex   float64
	BaselineK         float64
	// LastHabitableTurn is the final campaign turn on which this tile has any
	// capacity at all, or -1 for a tile that never does. Habitability is a pure
	// function of tile and turn — the seed only perturbs local temperature, not
	// the vegetation index BaselineK is built from — so the whole trajectory is
	// known at world generation. The HUD needs it to tell a tile that is closed
	// for this cold snap apart from one that is finished for the campaign.
	LastHabitableTurn  int
	EcologicalK        float64
	Degradation        float64
	FloraStock         float64
	FloraCap           float64
	FaunaStock         float64
	FaunaCap           float64
	WaterStock         float64
	WaterCap           float64
	Fauna              FaunaSummary
	VisibleMacroImpact MacroImpactSummary
}

type FaunaSummary struct {
	Weights            [FaunaGroupCount]float64
	HuntingSupported   bool
	MegafaunaSupported bool
}

type MacroImpactSummary struct {
	Visible        bool
	Episode        MacroEpisode
	ResourceFactor float64
	HabitatFactor  float64
}

type Band struct {
	ID                          BandID
	Species                     Species
	TileID                      TileID
	Population                  uint32
	Health                      float64
	StoredFood                  float64
	AllocationBP                [AssignmentCount]uint16
	AcquiredTech                uint16
	ResearchTarget              Tech
	HasResearchTarget           bool
	ResearchProgress            [TechCount]float64
	ResearchOptions             [TechCount]ResearchOption
	HeritableState              [HeritableTraitCount]float64
	LastFoodReport              FoodTurnReport
	LastMortality               MortalityReport
	LastOutcomeReport           OutcomeReport
	SpatialActionUsed           bool
	QueuedMigration             TileID
	HasQueuedMigration          bool
	OriginalResearchGainPreview float64
	MigrationCandidates         []MigrationCandidate
	InterbreedCandidateIDs      []BandID
	// HasInterbreedTarget and InterbreedTargetID expose the intent the band has
	// already accepted this turn, so presentation can show what the player
	// chose before the turn resolves it rather than leaving the action silent.
	HasInterbreedTarget   bool
	InterbreedTargetID    BandID
	PassageStatuses       [PassageCount]PassageStatus
	Stress                float64
	SeasonalMortalityRate float64
	ChronicMortalityRate  float64
}

// ResearchOption is a projected view of the authoritative prerequisite DAG.
// Available excludes already learned technologies.
type ResearchOption struct {
	Available        bool
	Acquired         bool
	Current          bool
	Cost             float64
	PrerequisiteMask uint16
}

type FoodTurnReport struct {
	Turn       int
	RequiredFU float64
	DeficitFU  float64
}

func (r FoodTurnReport) ConsumedFU() float64 { return r.RequiredFU - r.DeficitFU }

func (r FoodTurnReport) DeficitFraction() float64 {
	if r.RequiredFU <= 0 {
		return 0
	}
	return r.DeficitFU / r.RequiredFU
}

type MortalityReport struct {
	Starvation float64
	Seasonal   float64
	Chronic    float64
	Macro      float64
	Acute      float64
}

type OutcomeReport struct {
	Turn                    int
	StartingPopulation      uint32
	EndingPopulation        uint32
	Growth                  float64
	StartingHealth          float64
	EndingHealth            float64
	NutritionDelta          float64
	WaterHealthLoss         float64
	DiseaseHealthLoss       float64
	GeneticBurdenHealthLoss float64
	MacroHealthLoss         float64
	AcuteDiseaseHealthLoss  float64
}

type MigrationCandidate struct {
	TileID                  TileID
	Cost                    float64
	Attraction              float64
	EcologicalK             float64
	UsableFoodEquivalent    float64
	WaterSurvivalEquivalent float64
	DestinationPopulation   uint64
	WarningSuitability      float64
	SeasonalMortalityRate   float64
	ChronicMortalityRate    float64
	// CrowdingDecline is the people this band would lose to crowding on its
	// first turn at the destination, or zero if the tile has room for it.
	CrowdingDecline float64
	Passage         PassageID
	RequiresPassage bool
}

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

func (v EventKind) String() string {
	return enumString(int(v), []string{"Migration", "Split", "Technology", "Interbreeding", "Acute Incident", "Macro Episode", "Achievement", "Extinction"}, "EventKind")
}

type Event struct {
	Turn    int
	Kind    EventKind
	BandID  BandID
	TileID  TileID
	Region  Region
	Summary string
}
