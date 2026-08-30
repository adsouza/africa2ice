package domain

const (
	MaxBands                            = 256
	MaxArchaicBands                     = 96
	AllocationBasisPoints               = 10_000
	FoodStorageTurns                    = 3.0
	FoodSpoilageRate                    = 0.10
	MinSplitSourcePopulation Population = 40
)

type FoodTurnReport struct {
	Turn       int
	RequiredFU float64
	DeficitFU  float64
}

type MortalityReport struct {
	Starvation float64
	Seasonal   float64
	Chronic    float64
	Macro      float64
	Acute      float64
}

// OutcomeReport preserves the bounded actuals needed to explain the most
// recently completed turn. It is historical display data, never a simulation
// input.
type OutcomeReport struct {
	Turn                    int
	StartingPopulation      Population
	EndingPopulation        Population
	Growth                  float64
	StartingHealth          Health
	EndingHealth            Health
	NutritionDelta          float64
	WaterHealthLoss         float64
	DiseaseHealthLoss       float64
	GeneticBurdenHealthLoss float64
	MacroHealthLoss         float64
	AcuteDiseaseHealthLoss  float64
}

type Band struct {
	ID                  BandID
	Species             Species
	TileID              TileID
	Population          Population
	Health              Health
	StoredFood          FU
	Allocation          [AssignmentCount]AssignmentBP
	Technology          TechnologyState
	Heritable           HeritableState
	LastFoodReport      FoodTurnReport
	LastMortality       MortalityReport
	LastOutcomeReport   OutcomeReport
	SpatialActionUsed   bool
	QueuedMigration     TileID
	QueuedOrigin        TileID
	QueuedPassage       PassageID
	QueueUsesPassage    bool
	HasQueuedMigration  bool
	InterbreedTarget    BandID
	HasInterbreedTarget bool
}

func (band Band) Workers(role WorkforceRole) float64 {
	if role >= AssignmentCount {
		return 0
	}
	share := float64(band.Allocation[role]) / AllocationBasisPoints
	return float64(float64(band.Population) * share)
}

func FoodStorageCapacity(population Population) float64 {
	if population <= 0 {
		return 0
	}
	return float64(float64(population) * FoodStorageTurns)
}

func (band *Band) SetAssignment(allocation [AssignmentCount]AssignmentBP) error {
	if err := ValidateAssignments(allocation); err != nil {
		return err
	}
	band.Allocation = allocation
	return nil
}

func splitBand(parent Band, childID BandID) (Band, Band, error) {
	if parent.Population < MinSplitSourcePopulation {
		return Band{}, Band{}, ErrSplitPopulationTooLow
	}
	childPopulation := parent.Population / 2
	sourcePopulation := parent.Population - childPopulation
	halfFood := FU(float64(parent.StoredFood) / 2)
	left, right := parent, parent
	left.Population, right.Population = sourcePopulation, childPopulation
	left.StoredFood, right.StoredFood = halfFood, halfFood
	right.ID = childID
	left.LastFoodReport, right.LastFoodReport = FoodTurnReport{}, FoodTurnReport{}
	left.LastMortality, right.LastMortality = MortalityReport{}, MortalityReport{}
	left.LastOutcomeReport, right.LastOutcomeReport = OutcomeReport{}, OutcomeReport{}
	left.SpatialActionUsed, right.SpatialActionUsed = true, true
	left.HasQueuedMigration, right.HasQueuedMigration = false, false
	left.HasInterbreedTarget, right.HasInterbreedTarget = false, false
	return left, right, nil
}
