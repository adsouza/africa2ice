package application

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

// DESIGN.md §12 step 7 requires algorithm coverage to come from executable
// schema rather than from parsing the design document: one case per exported
// SaveState field whose name ends in "Algorithm", checked against the struct by
// reflection so that adding a field without a case fails the build's tests.

type algorithmCase struct {
	// field is the exact exported SaveState field name this case covers.
	field string
	// current is the identifier this executable writes today.
	current string
	// olderSupported lists identifiers an older save may legitimately carry.
	// Every list is empty while v1 is unreleased: there is no released
	// identifier to migrate from, and inventing one would test fiction.
	olderSupported []string
	// unsupported is one identifier the executable must refuse.
	unsupported string
	// probe extracts exactly the derived state this algorithm determines. The
	// shared suite proves the probe is identical across a save/load round trip,
	// which is the future-equivalence invariant stated per algorithm.
	probe func(world *domain.World) string
}

func digest(values ...any) string {
	hash := fnv.New64a()
	for _, value := range values {
		_, _ = fmt.Fprintf(hash, "%v|", value)
	}
	return fmt.Sprintf("%016x", hash.Sum64())
}

// eachLandTile folds a per-tile probe over the map in stable tile order.
func eachLandTile(world *domain.World, extract func(domain.TileID, domain.TileGeography, domain.HabitatTile, domain.TileState) []any) string {
	grid, habitat, states := world.Grid(), world.Habitat(), world.TileStates()
	values := make([]any, 0, domain.TileCount)
	for id := range domain.TileCount {
		geography, ok := grid.Tile(domain.TileID(id))
		if !ok {
			values = append(values, "absent")
			continue
		}
		values = append(values, extract(domain.TileID(id), geography, habitat[id], states[id])...)
	}
	return digest(values...)
}

func eachBand(world *domain.World, extract func(domain.Band) []any) string {
	values := make([]any, 0, 16)
	for _, band := range world.Bands() {
		values = append(values, extract(band)...)
	}
	return digest(values...)
}

// bandContext resolves the geography, habitat, and season a band-scoped probe
// needs, so each case below stays a single expression about its own algorithm.
func bandContext(world *domain.World, band domain.Band) (domain.TileGeography, domain.HabitatTile, domain.Season) {
	geography, _ := world.Grid().Tile(band.TileID)
	season, _ := domain.SeasonForTurn(world.Turn())
	return geography, world.Habitat()[band.TileID], season
}

var algorithmCases = []algorithmCase{{
	field: "GeographyAlgorithm", current: "dispersal-map-v4", unsupported: "dispersal-map-v3",
	probe: func(world *domain.World) string {
		return eachLandTile(world, func(id domain.TileID, geography domain.TileGeography, _ domain.HabitatTile, _ domain.TileState) []any {
			return []any{geography.Land, geography.Region, geography.X, geography.Y, geography.ElevationKm, geography.BaseMoisture}
		})
	},
}, {
	field: "NaturalShelterMaskAlgorithm", current: "authored-ellipse-v1", unsupported: "raster-shelter-v2",
	probe: func(world *domain.World) string {
		return eachLandTile(world, func(_ domain.TileID, geography domain.TileGeography, _ domain.HabitatTile, _ domain.TileState) []any {
			return []any{geography.NaturalShelter}
		})
	},
}, {
	field: "CampaignClockAlgorithm", current: "four-era-v1", unsupported: "five-era-v2",
	probe: func(world *domain.World) string {
		date, err := domain.CampaignDate(world.Turn())
		season, seasonErr := domain.SeasonForTurn(world.Turn())
		return digest(date.Turn, date.YearBP, date.Era, date.CalendarProgress, season, err, seasonErr)
	},
}, {
	field: "ClimateAlgorithm", current: "hybrid-abrupt-moisture-v1", unsupported: "linear-cooling-v2",
	probe: func(world *domain.World) string {
		climate := world.Climate()
		return digest(climate.Turn, climate.LongTermTempOffset, climate.SeasonalTempOffset, climate.ClimateNoise,
			climate.GlobalTempOffset, climate.LongTermMoistureOffset, climate.AridityIndex, climate.Epoch, climate.RegionalAbrupt)
	},
}, {
	field: "TemperatureAlgorithm", current: "lat-elev-offset-v1", unsupported: "lat-elev-offset-v2",
	probe: func(world *domain.World) string {
		return eachLandTile(world, func(_ domain.TileID, _ domain.TileGeography, habitat domain.HabitatTile, _ domain.TileState) []any {
			return []any{habitat.HabitatTemperatureC, habitat.LocalTemperatureC}
		})
	},
}, {
	field: "MacroEventAlgorithm", current: "bounded-regional-v1", unsupported: "global-episode-v2",
	probe: func(world *domain.World) string {
		turn := world.Turn()
		return eachLandTile(world, func(_ domain.TileID, geography domain.TileGeography, _ domain.HabitatTile, _ domain.TileState) []any {
			impact := domain.MacroImpactAt(geography, turn)
			return []any{impact.Active, impact.Zone, impact.Intensity, impact.FloraFactor, impact.FaunaFactor,
				impact.WaterFactor, impact.HabitatFactor, impact.LossFraction, impact.HealthLoss}
		})
	},
}, {
	field: "ExplorationAlgorithm", current: "sapiens-frontier-v1", unsupported: "omniscient-v2",
	probe: func(world *domain.World) string {
		values := make([]any, 0, domain.TileCount)
		for id := range domain.TileCount {
			values = append(values, world.IsExplored(domain.TileID(id)))
		}
		return digest(values...)
	},
}, {
	field: "BandAlgorithm", current: "capacity-safe-half-global-cap-v2", unsupported: "dynamic-cap-v3",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.ID, band.Species, band.TileID, band.Population, band.Health, band.StoredFood}
		})
	},
}, {
	field: "ArchaicPolicyAlgorithm", current: "ranked-pressure-v1", unsupported: "random-walk-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			if band.Species != domain.ArchaicHominin {
				return nil
			}
			return []any{band.ID, band.TileID, band.Allocation, band.HasQueuedMigration, band.QueuedMigration}
		})
	},
}, {
	field: "AssignmentAlgorithm", current: "proportional-basis-points-v1", unsupported: "percentage-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			workers := make([]any, 0, domain.AssignmentCount)
			for role := range domain.AssignmentCount {
				workers = append(workers, band.Workers(domain.WorkforceRole(role)))
			}
			return append([]any{band.Allocation}, workers...)
		})
	},
}, {
	field: "FoodStorageAlgorithm", current: "population-food-turns-v1", unsupported: "flat-cap-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.StoredFood, domain.FoodStorageCapacity(band.Population)}
		})
	},
}, {
	field: "FoodConversionAlgorithm", current: "normalized-source-v1", unsupported: "raw-mass-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			metabolism := float64(band.Heritable[domain.FattyAcidMetabolism])
			return []any{
				domain.FattyAcidConversion(domain.PlantFood, metabolism),
				domain.FattyAcidConversion(domain.AnimalFood, metabolism),
				domain.FattyAcidConversion(domain.AquaticFood, metabolism),
			}
		})
	},
}, {
	field: "ForagingAlgorithm", current: "linear-shared-flora-v1", unsupported: "saturating-flora-v2",
	probe: func(world *domain.World) string {
		habitat := world.Habitat()
		return eachBand(world, func(band domain.Band) []any {
			return []any{domain.ForagingRate(band, habitat[band.TileID].Biome)}
		})
	},
}, {
	field: "HuntingAlgorithm", current: "linear-shared-fauna-v1", unsupported: "stochastic-fauna-v2",
	probe: func(world *domain.World) string {
		habitat := world.Habitat()
		return eachBand(world, func(band domain.Band) []any {
			geography, _ := world.Grid().Tile(band.TileID)
			profile, _ := domain.FaunaFor(geography.Region, habitat[band.TileID].Biome, habitat[band.TileID].BaselineK > 0)
			rates := make([]any, 0, domain.FaunaGroupCount)
			for group := domain.FaunaGroup(0); group < domain.FaunaGroupCount; group++ {
				rates = append(rates, domain.GroupHuntingRate(band, geography.Region, profile, group))
			}
			return rates
		})
	},
}, {
	field: "MegafaunaAlgorithm", current: "linear-megafauna-v1", unsupported: "herd-tracking-v2",
	probe: func(world *domain.World) string {
		habitat := world.Habitat()
		return eachBand(world, func(band domain.Band) []any {
			geography, _ := world.Grid().Tile(band.TileID)
			profile, _ := domain.FaunaFor(geography.Region, habitat[band.TileID].Biome, habitat[band.TileID].BaselineK > 0)
			return []any{profile.MegafaunaSupported, domain.GroupHuntingRate(band, geography.Region, profile, domain.Megafauna)}
		})
	},
}, {
	field: "HuntingRiskAlgorithm", current: "linear-share-work-risk-v1", unsupported: "flat-risk-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			geography, habitat, season := bandContext(world, band)
			workRisk := [domain.AcuteKindCount]float64{}
			workRisk[domain.AcutePredation] = 0.04
			return []any{domain.AcuteProbabilities(band, geography, habitat, season, workRisk, false, domain.PassageCount, 0)}
		})
	},
}, {
	field: "FaunaProfileAlgorithm", current: "region-biome-v1", unsupported: "biome-only-v2",
	probe: func(world *domain.World) string {
		return eachLandTile(world, func(_ domain.TileID, geography domain.TileGeography, habitat domain.HabitatTile, _ domain.TileState) []any {
			profile, ok := domain.FaunaFor(geography.Region, habitat.Biome, habitat.BaselineK > 0)
			return []any{ok, profile.Archetype, profile.Weights, profile.HuntingSupported, profile.MegafaunaSupported}
		})
	},
}, {
	field: "ShelterAlgorithm", current: "saturating-share-terrain-v1", unsupported: "linear-shelter-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			geography, _, _ := bandContext(world, band)
			share := float64(band.Allocation[domain.Shelter]) / domain.AllocationBasisPoints
			return []any{domain.ShelterCurve(share, geography.NaturalShelter), domain.ShelterCurve(share, 0)}
		})
	},
}, {
	field: "TechnologyOwnershipAlgorithm", current: "band-local-v1", unsupported: "species-global-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.Technology.Acquired, band.Technology.Target, band.Technology.HasTarget}
		})
	},
}, {
	field: "ResearchProductionAlgorithm", current: "saturating-toolcraft-v1", unsupported: "linear-toolcraft-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.Technology.Progress}
		})
	},
}, {
	field: "KnowledgeContactAlgorithm", current: "co-located-cross-species-v1", unsupported: "adjacent-contact-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.TileID, band.Species, band.HasInterbreedTarget, band.InterbreedTarget}
		})
	},
}, {
	field: "KnowledgeDiffusionAlgorithm", current: "stacked-acquired-v1", unsupported: "single-source-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.Technology.Acquired, band.Technology.Progress, band.TileID}
		})
	},
}, {
	field: "HeritableStateAlgorithm", current: "band-six-trait-v1", unsupported: "band-eight-trait-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.Heritable, domain.HeritableTraitCount}
		})
	},
}, {
	field: "GeneticSelectionAlgorithm", current: "trait-functions-v1", unsupported: "fitness-matrix-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			geography, habitat, season := bandContext(world, band)
			return []any{domain.SelectionDeltas(band, geography, habitat, season, 0.5)}
		})
	},
}, {
	field: "GeneFlowAlgorithm", current: "local-reciprocal-v1", unsupported: "global-mixing-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			return []any{band.TileID, band.Species, band.Population, band.Heritable}
		})
	},
}, {
	field: "MutationAlgorithm", current: "rare-emergence-v1", unsupported: "per-turn-drift-v2",
	probe: func(world *domain.World) string {
		state, err := world.ExportState()
		return digest(state.RNGState, err, eachBand(world, func(band domain.Band) []any {
			return []any{band.Heritable}
		}))
	},
}, {
	field: "MovementAlgorithm", current: "eight-way-escarpment-corners-v2", unsupported: "eight-way-no-water-corners-v1",
	probe: func(world *domain.World) string {
		grid := world.Grid()
		return eachBand(world, func(band domain.Band) []any {
			edges := make([]any, 0, domain.MaxGridNeighbors)
			for _, edge := range grid.OrdinaryEdges(band.TileID) {
				edges = append(edges, edge.To, edge.StepLength)
			}
			return edges
		})
	},
}, {
	field: "MovementCostAlgorithm", current: "destination-vegetation-v1", unsupported: "slope-weighted-v2",
	probe: func(world *domain.World) string {
		return eachLandTile(world, func(_ domain.TileID, _ domain.TileGeography, habitat domain.HabitatTile, _ domain.TileState) []any {
			return []any{habitat.MovementCost}
		})
	},
}, {
	field: "PassageAlgorithm", current: "named-asymmetric-v1", unsupported: "symmetric-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			statuses := make([]any, 0, domain.PassageCount)
			for passage := domain.PassageID(0); passage < domain.PassageCount; passage++ {
				statuses = append(statuses, world.PassageStatus(band.ID, passage))
			}
			return statuses
		})
	},
}, {
	field: "ResourceAlgorithm", current: "toward-cap-v1", unsupported: "instant-refill-v2",
	probe: func(world *domain.World) string {
		season, _ := domain.SeasonForTurn(world.Turn())
		return eachLandTile(world, func(_ domain.TileID, _ domain.TileGeography, habitat domain.HabitatTile, state domain.TileState) []any {
			caps := domain.ResourceCaps(habitat.Biome, season, state.Degradation, habitat.BaselineK)
			return []any{state.Degradation, state.Stock, caps}
		})
	},
}, {
	field: "HazardAlgorithm", current: "split-v1", unsupported: "combined-v2",
	probe: func(world *domain.World) string {
		return eachBand(world, func(band domain.Band) []any {
			geography, habitat, season := bandContext(world, band)
			seasonal, chronic := domain.Phase3MortalityRates(band, geography, habitat, season)
			disease, genetic := domain.Phase3HealthLosses(band, geography, habitat, season)
			return []any{seasonal, chronic, disease, genetic}
		})
	},
}, {
	field: "KinSupportAlgorithm", current: "saturating-kin-acute-v1", unsupported: "linear-kin-contact-v2",
	probe: func(world *domain.World) string {
		// The remaining-risk factor each band derives from its living
		// same-species neighbours, recomputed from published state.
		bands := world.Bands()
		grid := world.Grid()
		values := make([]any, 0, len(bands))
		for _, band := range bands {
			contacts := 0
			for _, other := range bands {
				if other.ID == band.ID || other.Population == 0 || band.Population == 0 || other.Species != band.Species {
					continue
				}
				inContact := other.TileID == band.TileID
				for _, edge := range grid.OrdinaryEdges(band.TileID) {
					if edge.To == other.TileID {
						inContact = true
					}
				}
				if inContact {
					contacts++
				}
			}
			values = append(values, band.ID, contacts, domain.KinSupportRemainingRisk(contacts))
		}
		return digest(values...)
	},
}, {
	field: "PopulationRoundingAlgorithm", current: "stochastic-v1", unsupported: "nearest-integer-v2",
	probe: func(world *domain.World) string {
		// The rounding rule's output is the integral population itself, taken
		// together with the RNG state its draws advance.
		state, err := world.ExportState()
		return digest(state.RNGState, err, eachBand(world, func(band domain.Band) []any {
			return []any{band.Population, domain.MaxPopulation}
		}))
	},
}, {
	field: "RNGAlgorithm", current: "pcg-splitmix-v1", unsupported: "xoshiro-v2",
	probe: func(world *domain.World) string {
		state, err := world.ExportState()
		return digest(state.RNGState, err)
	},
}}

func algorithmFieldNames(t *testing.T) []string {
	t.Helper()
	structType := reflect.TypeOf(AlgorithmVersions{})
	names := make([]string, 0, structType.NumField())
	for index := range structType.NumField() {
		field := structType.Field(index)
		if field.Type.Kind() != reflect.String || !strings.HasSuffix(field.Name, "Algorithm") {
			continue
		}
		names = append(names, field.Name)
	}
	sort.Strings(names)
	return names
}

// TestAlgorithmCaseTableMatchesSaveStateSchema is the reflection sentinel: the
// case table and the wire struct must name exactly the same set, with no
// missing, extra, or duplicate case.
func TestAlgorithmCaseTableMatchesSaveStateSchema(t *testing.T) {
	fields := algorithmFieldNames(t)
	covered := make(map[string]int, len(algorithmCases))
	for _, testCase := range algorithmCases {
		covered[testCase.field]++
	}
	var missing, duplicated []string
	for _, name := range fields {
		switch covered[name] {
		case 1:
		case 0:
			missing = append(missing, name)
		default:
			duplicated = append(duplicated, fmt.Sprintf("%s (%d cases)", name, covered[name]))
		}
		delete(covered, name)
	}
	extra := make([]string, 0, len(covered))
	for name := range covered {
		extra = append(extra, name)
	}
	sort.Strings(extra)
	if len(missing) != 0 {
		t.Errorf("SaveState algorithm fields with no case (%d): %s", len(missing), strings.Join(missing, ", "))
	}
	if len(extra) != 0 {
		t.Errorf("cases naming no SaveState field: %s", strings.Join(extra, ", "))
	}
	if len(duplicated) != 0 {
		t.Errorf("duplicate cases: %s", strings.Join(duplicated, ", "))
	}
}

// TestEveryCurrentAlgorithmIdentifierMatchesTheWire proves the table states the
// identifiers the executable actually writes, so a silently bumped version in
// supportedAlgorithms cannot pass unnoticed.
func TestEveryCurrentAlgorithmIdentifierMatchesTheWire(t *testing.T) {
	written := reflect.ValueOf(supportedAlgorithms)
	for _, testCase := range algorithmCases {
		field := written.FieldByName(testCase.field)
		if !field.IsValid() {
			t.Errorf("%s: no such SaveState field", testCase.field)
			continue
		}
		if field.String() != testCase.current {
			t.Errorf("%s: wire writes %q, case declares %q", testCase.field, field.String(), testCase.current)
		}
	}
}

// TestEveryCurrentAlgorithmRoundTripsAndReconstructs is the future-equivalence
// pass: what each algorithm derives must be reconstructed identically from
// saved state plus versioned configuration rather than serialized.
func TestEveryCurrentAlgorithmRoundTripsAndReconstructs(t *testing.T) {
	service, err := NewGameService(0x9e3779b97f4a7c15)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := service.EndTurn(); err != nil {
			t.Fatal(err)
		}
	}
	save, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSaveState(save)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSaveState(encoded)
	if err != nil {
		t.Fatal(err)
	}
	after, err := decoded.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range algorithmCases {
		t.Run(testCase.field, func(t *testing.T) {
			if testCase.probe == nil {
				t.Fatal("case supplies no reconstruction probe")
			}
			original, restored := testCase.probe(service.world), testCase.probe(after)
			if original != restored {
				t.Fatalf("%s: derived state differs across the wire: %s vs %s", testCase.field, original, restored)
			}
		})
	}
}

// TestUnsupportedAlgorithmIdentifiersAreRejected proves a save written by a
// different implementation of any one algorithm is refused, and that refusing
// it leaves the running world in place.
func TestUnsupportedAlgorithmIdentifiersAreRejected(t *testing.T) {
	for _, testCase := range algorithmCases {
		t.Run(testCase.field, func(t *testing.T) {
			repository := &repositoryStub{writable: true}
			service, err := NewGameServiceWithRepository(7, repository)
			if err != nil {
				t.Fatal(err)
			}
			before, err := service.ExportSaveState()
			if err != nil {
				t.Fatal(err)
			}
			liveWorld := service.world
			liveWorldRevision := service.worldRevision
			liveTerrainRevision := service.terrainRevision
			if testCase.unsupported == testCase.current {
				t.Fatal("the unsupported identifier must differ from the current one")
			}
			tampered := before
			field := reflect.ValueOf(&tampered.AlgorithmVersions).Elem().FieldByName(testCase.field)
			if !field.IsValid() || !field.CanSet() {
				t.Fatalf("%s is not settable on the wire struct", testCase.field)
			}
			field.SetString(testCase.unsupported)
			repository.written = &tampered
			operationID, err := service.BeginLoad(int(Manual1))
			if err != nil {
				t.Fatal(err)
			}
			results := service.PollStorage()
			if len(results) != 1 || results[0].OperationID != operationID || results[0].Err == nil {
				t.Fatalf("rejected load result = %#v", results)
			}
			if results[0].ReplacementFrame != nil {
				t.Fatal("a rejected load published a replacement frame")
			}
			if service.world != liveWorld || service.worldRevision != liveWorldRevision || service.terrainRevision != liveTerrainRevision {
				t.Fatal("a rejected load replaced or revised the running world")
			}
			after, err := service.ExportSaveState()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, before) {
				t.Fatal("a rejected load mutated the running save state")
			}
		})
	}
}

// TestNoMigrationIsInventedBeforeV1Ships guards the one way this table could
// lie: declaring support for an older identifier that was never released.
func TestNoMigrationIsInventedBeforeV1Ships(t *testing.T) {
	for _, testCase := range algorithmCases {
		if len(testCase.olderSupported) == 0 {
			continue
		}
		for _, older := range testCase.olderSupported {
			t.Errorf("%s declares older identifier %q, but v1 is unreleased so no save can carry it", testCase.field, older)
		}
	}
}

// TestSchemaVersionIsOrderedNotOpaque keeps the schema check separate from the
// algorithm strings: schema versions compare as ordered integers, while
// algorithm identifiers are opaque and only ever supported or not.
func TestSchemaVersionIsOrderedNotOpaque(t *testing.T) {
	service, err := NewGameService(11)
	if err != nil {
		t.Fatal(err)
	}
	save, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	if save.SchemaVersion != SaveSchemaVersion {
		t.Fatalf("exported schema %d, want %d", save.SchemaVersion, SaveSchemaVersion)
	}
	newer := save
	newer.SchemaVersion = SaveSchemaVersion + 1
	if _, err := newer.RestoreWorld(); err == nil {
		t.Fatal("a schema newer than the executable was accepted")
	}
	if _, err := save.RestoreWorld(); err != nil {
		t.Fatalf("the current schema must still load: %v", err)
	}
}
