package gameapi

import "strconv"

type Species uint8

const (
	HomoSapiens Species = iota
	ArchaicHominin
	SpeciesCount
)

func (v Species) String() string {
	return enumString(int(v), []string{"Homo sapiens", "Archaic hominin"}, "Species")
}

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

func (v Biome) String() string {
	return enumString(int(v), []string{"Riverine Woodland", "Savanna", "Coastal Shrubland", "Mountainous Highlands", "Semi-Arid Desert", "Glacial Tundra"}, "Biome")
}

type Season uint8

const (
	SeasonWarm Season = iota
	SeasonCooling
	SeasonCold
	SeasonWarming
	SeasonCount
)

func (v Season) String() string {
	return enumString(int(v), []string{"Warm", "Cooling", "Cold", "Warming"}, "Season")
}

type CampaignEra uint8

const (
	EraEarly CampaignEra = iota
	EraMiddle
	EraLate
	EraFinal
	CampaignEraCount
)

func (v CampaignEra) String() string {
	return enumString(int(v), []string{"Early", "Middle", "Late", "Final"}, "CampaignEra")
}

type Tech uint8

const (
	Firecraft Tech = iota
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

func (v Tech) String() string {
	return enumString(int(v), []string{"Firecraft", "Hafted Tools", "Plant Knowledge", "Tailored Clothing", "Cordage and Nets", "Campcraft", "Medicinal Knowledge", "Trapping", "Coastal Navigation"}, "Tech")
}

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

func (v HeritableTrait) String() string {
	return enumString(int(v), []string{"Cold Adaptation", "High-Altitude Adaptation", "Innate Immune Reactivity", "Arid-Climate Adaptation", "Pigmentation Level", "Fatty-Acid Metabolism"}, "HeritableTrait")
}

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

func (v Region) String() string {
	return enumString(int(v), []string{"East Africa", "Rest of Africa", "Arabia", "Levant", "Frangistan", "Central Asia", "South Asia", "Southeast Asia", "East Asia", "Yellow River Basin", "Sahul", "Siberia", "Beringia"}, "Region")
}

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

func (v FaunaGroup) String() string {
	return enumString(int(v), []string{"Small Game", "Medium Game", "Large Game", "Megafauna", "Inshore Aquatic", "Pelagic Aquatic"}, "FaunaGroup")
}

type WorkforceRole uint8

const (
	Foraging WorkforceRole = iota
	HuntingAndFishing
	Toolcraft
	MegafaunaTracking
	Shelter
	AssignmentCount
)

func (v WorkforceRole) String() string {
	return enumString(int(v), []string{"Foraging", "Hunting and Fishing", "Toolcraft", "Megafauna Tracking", "Shelter"}, "WorkforceRole")
}

type PassageID uint8

const (
	NorthWallacea PassageID = iota
	SouthWallacea
	BeringStrait
	PassageCount
)

func (v PassageID) String() string {
	return enumString(int(v), []string{"North Wallacea", "South Wallacea", "Bering Strait"}, "PassageID")
}

type PassageStatus uint8

const (
	PassageUnavailable PassageStatus = iota
	PassageLocked
	PassageOpen
	PassageStatusCount
)

func (v PassageStatus) String() string {
	return enumString(int(v), []string{"Unavailable", "Locked", "Open"}, "PassageStatus")
}

type MacroEpisode uint8

const (
	CampanianIgnimbrite MacroEpisode = iota
	MacroEpisodeCount
)

func (v MacroEpisode) String() string {
	return enumString(int(v), []string{"Campanian Ignimbrite"}, "MacroEpisode")
}

type CampaignResult uint8

const (
	Ongoing CampaignResult = iota
	Victory
	Extinction
	DispersalFailed
	CampaignResultCount
)

func (v CampaignResult) String() string {
	return enumString(int(v), []string{"Ongoing", "Victory", "Extinction", "Dispersal Failed"}, "CampaignResult")
}

func enumString(value int, names []string, kind string) string {
	if value >= 0 && value < len(names) {
		return names[value]
	}
	return kind + "(" + strconv.Itoa(value) + ")"
}
