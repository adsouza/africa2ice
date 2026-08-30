package domain

type FaunaArchetype uint8

const (
	OpenMixed FaunaArchetype = iota
	WoodlandRiverine
	CoastalMixed
	AridSmallGame
	HighlandMixed
	ColdSteppe
	TropicalIsland
	BeringianCoast
	FaunaArchetypeCount
)

type FaunaProfile struct {
	Archetype          FaunaArchetype
	Weights            [FaunaGroupCount]float64
	HuntingSupported   bool
	MegafaunaSupported bool
}

var faunaProfiles = [FaunaArchetypeCount]FaunaProfile{
	{OpenMixed, [FaunaGroupCount]float64{0.15, 0.25, 0.30, 0.20, 0.10, 0.00}, true, true},
	{WoodlandRiverine, [FaunaGroupCount]float64{0.20, 0.20, 0.15, 0.10, 0.30, 0.05}, true, true},
	{CoastalMixed, [FaunaGroupCount]float64{0.15, 0.15, 0.10, 0.05, 0.35, 0.20}, true, true},
	{AridSmallGame, [FaunaGroupCount]float64{0.40, 0.30, 0.15, 0.00, 0.15, 0.00}, true, false},
	{HighlandMixed, [FaunaGroupCount]float64{0.25, 0.30, 0.20, 0.15, 0.10, 0.00}, true, true},
	{ColdSteppe, [FaunaGroupCount]float64{0.10, 0.20, 0.30, 0.35, 0.05, 0.00}, true, true},
	{TropicalIsland, [FaunaGroupCount]float64{0.20, 0.15, 0.10, 0.10, 0.25, 0.20}, true, true},
	{BeringianCoast, [FaunaGroupCount]float64{0.10, 0.20, 0.25, 0.30, 0.10, 0.05}, true, true},
}

var huntingIndex = [RegionCount]float64{1.00, 1.00, 0.65, 0.95, 1.15, 0.80, 1.05, 0.85, 0.95, 1.00, 0.90, 1.25, 1.15}

func FaunaFor(region Region, biome Biome, habitable bool) (FaunaProfile, bool) {
	if !habitable {
		return FaunaProfile{}, true
	}
	if region >= RegionCount || biome >= BiomeCount {
		return FaunaProfile{}, false
	}
	archetype := baselineArchetype(biome)
	switch {
	case region == Arabia && biome == Savanna:
		archetype = AridSmallGame
	case (region == SoutheastAsia || region == Sahul) && biome == CoastalShrubland:
		archetype = TropicalIsland
	case region == Beringia && (biome == CoastalShrubland || biome == GlacialTundra):
		archetype = BeringianCoast
	}
	return faunaProfiles[archetype], true
}

func baselineArchetype(biome Biome) FaunaArchetype {
	switch biome {
	case Savanna:
		return OpenMixed
	case RiverineWoodland:
		return WoodlandRiverine
	case CoastalShrubland:
		return CoastalMixed
	case SemiAridDesert:
		return AridSmallGame
	case MountainousHighlands:
		return HighlandMixed
	case GlacialTundra:
		return ColdSteppe
	default:
		return OpenMixed
	}
}

func HuntingIndex(region Region) (float64, bool) {
	if region >= RegionCount {
		return 0, false
	}
	return huntingIndex[region], true
}

var archaicAssignments = [FaunaArchetypeCount][AssignmentCount]AssignmentBP{
	{3000, 2500, 1200, 1500, 1800},
	{3000, 3000, 1200, 800, 2000},
	{2500, 3500, 1200, 500, 2300},
	{3000, 4000, 1200, 0, 1800},
	{2800, 2800, 1200, 1000, 2200},
	{800, 3200, 1500, 2000, 2500},
	{2200, 3800, 1200, 800, 2000},
	{800, 3400, 1500, 1800, 2500},
}

func ArchaicAssignment(region Region, biome Biome) ([AssignmentCount]AssignmentBP, bool) {
	profile, ok := FaunaFor(region, biome, true)
	if !ok {
		return [AssignmentCount]AssignmentBP{}, false
	}
	return archaicAssignments[profile.Archetype], true
}
