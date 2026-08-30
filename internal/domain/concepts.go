package domain

type Species uint8

const (
	HomoSapiens Species = iota
	ArchaicHominin
	SpeciesCount
)

type Biome uint8

const (
	RiverineWoodland Biome = iota
	Savanna
	CoastalShrubland
	MountainousHighlands
	SemiAridDesert
	GlacialTundra
	BiomeCount
)

type Season uint8

const (
	SeasonWarm Season = iota
	SeasonCooling
	SeasonCold
	SeasonWarming
	SeasonCount
)

type CampaignEra uint8

const (
	EraEarly CampaignEra = iota
	EraMiddle
	EraLate
	EraFinal
	CampaignEraCount
)

type Technology uint8

const (
	Firecraft Technology = iota
	HaftedTools
	PlantKnowledge
	TailoredClothing
	CordageAndNets
	Campcraft
	MedicinalKnowledge
	Trapping
	CoastalNavigation
	TechCount
)

type HeritableTrait uint8

const (
	ColdAdaptation HeritableTrait = iota
	HighAltitudeAdaptation
	InnateImmuneReactivity
	AridClimateAdaptation
	PigmentationLevel
	FattyAcidMetabolism
	HeritableTraitCount
)

type WorkforceRole uint8

const (
	Foraging WorkforceRole = iota
	HuntingAndFishing
	Toolcraft
	MegafaunaTracking
	Shelter
	AssignmentCount
)

type Region uint8

const (
	EastAfrica Region = iota
	RestOfAfrica
	Arabia
	Levant
	Frangistan
	CentralAsia
	SouthAsia
	SoutheastAsia
	EastAsia
	YellowRiverBasin
	Sahul
	Siberia
	Beringia
	RegionCount
)

type FaunaGroup uint8

const (
	SmallGame FaunaGroup = iota
	MediumGame
	LargeGame
	Megafauna
	InshoreAquatic
	PelagicAquatic
	FaunaGroupCount
)
