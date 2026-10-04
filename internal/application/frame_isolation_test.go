package application

import (
	"bytes"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// maximumWorkloadService is a GameService over the 256-band, fully explored
// turn-300 performance fixture: every frame slice is populated at its limit.
func maximumWorkloadService(t *testing.T) *GameService {
	t.Helper()
	payload, err := os.ReadFile("../../testdata/performance_profile_save.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	if service.world, err = state.RestoreWorld(); err != nil {
		t.Fatal(err)
	}
	return service
}

// mutateEverything changes every settable leaf reachable from value and
// appends to every slice, so any memory a frame shares with the domain or with
// another frame is overwritten somewhere. It returns how many leaves it
// changed.
func mutateEverything(value reflect.Value) int {
	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return 0
		}
		return mutateEverything(value.Elem())
	case reflect.Struct:
		changed := 0
		for index := range value.NumField() {
			if field := value.Field(index); field.CanSet() {
				changed += mutateEverything(field)
			}
		}
		return changed
	case reflect.Array:
		changed := 0
		for index := range value.Len() {
			changed += mutateEverything(value.Index(index))
		}
		return changed
	case reflect.Slice:
		changed := 0
		for index := range value.Len() {
			changed += mutateEverything(value.Index(index))
		}
		if value.CanSet() {
			value.Set(reflect.Append(value, reflect.Zero(value.Type().Elem())))
			changed++
		}
		return changed
	case reflect.Bool:
		value.SetBool(!value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(value.Int() + 1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(value.Uint() + 1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(value.Float() + 1)
	case reflect.String:
		value.SetString(value.String() + "~")
	default:
		return 0
	}
	return 1
}

// sliceBackings records the backing array of every non-empty slice reachable
// from value.
func sliceBackings(value reflect.Value, into map[uintptr]string, path string) {
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			sliceBackings(value.Elem(), into, path)
		}
	case reflect.Struct:
		for index := range value.NumField() {
			sliceBackings(value.Field(index), into, path+"."+value.Type().Field(index).Name)
		}
	case reflect.Array:
		for index := range value.Len() {
			sliceBackings(value.Index(index), into, path)
		}
	case reflect.Slice:
		if value.Cap() > 0 {
			into[value.Pointer()] = path
		}
		for index := range value.Len() {
			sliceBackings(value.Index(index), into, path)
		}
	}
}

func encodedWorld(t *testing.T, service *GameService) []byte {
	t.Helper()
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

// Step 6's no-recycling guarantee, asserted rather than assumed. Five frames
// are retained across four later turns, each with an independent twin taken at
// the same moment. Every leaf of the three oldest frames is then rewritten and
// every slice in them appended to. The domain must be byte-identical, the two
// newest frames must still equal their twins, and a fresh projection must
// equal the newest twin. Walking every field by reflection covers each field
// step 6 names and any added later.
func TestRetainedFramesShareNoMemoryWithTheDomainOrLaterFrames(t *testing.T) {
	service := maximumWorkloadService(t)
	var frames, twins []*gameapi.Frame
	for turn := range 5 {
		frame, err := service.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		twin, err := service.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(frame, twin) {
			t.Fatalf("two projections of turn %d differ before any mutation", frame.Turn)
		}
		frames, twins = append(frames, frame), append(twins, twin)
		if turn < 4 {
			if _, err := service.EndTurn(); err != nil {
				t.Fatal(err)
			}
		}
	}
	world := encodedWorld(t, service)

	for _, frame := range frames[:3] {
		if changed := mutateEverything(reflect.ValueOf(frame)); changed < 100_000 {
			t.Fatalf("mutated only %d leaves of a 256-band frame; the walk is not reaching the frame", changed)
		}
	}
	if reflect.DeepEqual(frames[0], twins[0]) {
		t.Fatal("mutation left the oldest frame equal to its twin; nothing was exercised")
	}
	if !bytes.Equal(encodedWorld(t, service), world) {
		t.Fatal("mutating retained frames changed the domain")
	}
	for index := 3; index < len(frames); index++ {
		if !reflect.DeepEqual(frames[index], twins[index]) {
			t.Fatalf("mutating older frames changed later frame %d", index)
		}
	}
	fresh, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fresh, twins[4]) {
		t.Fatal("a fresh projection no longer equals the newest twin")
	}
}

// Step 6's backing-address fixture: no slice published in one frame may share
// its backing array with a slice published in another.
func TestPublishedFramesShareNoSliceBackingArrays(t *testing.T) {
	service := maximumWorkloadService(t)
	owner := map[uintptr]int{}
	// The frames must stay reachable: a uintptr does not keep its slice alive,
	// and a collected frame's memory is free to reappear in the next one.
	var retained []*gameapi.Frame
	defer runtime.KeepAlive(&retained)
	for index := range 4 {
		var frame *gameapi.Frame
		var err error
		if index%2 == 0 {
			frame, err = service.Snapshot()
		} else {
			frame, err = service.EndTurn()
		}
		if err != nil {
			t.Fatal(err)
		}
		retained = append(retained, frame)
		backings := map[uintptr]string{}
		sliceBackings(reflect.ValueOf(frame), backings, "Frame")
		reached := map[string]bool{}
		for _, path := range backings {
			reached[path] = true
		}
		if !reached["Frame.Bands"] || !reached["Frame.Bands.MigrationCandidates"] || !reached["Frame.Tiles"] {
			t.Fatalf("frame %d walk reached %v; it must reach the band, candidate, and tile slices", index, reached)
		}
		for pointer, path := range backings {
			if previous, shared := owner[pointer]; shared {
				t.Fatalf("frame %d reuses the backing array of %s from frame %d", index, path, previous)
			}
			owner[pointer] = index
		}
	}
}

// Step 6's shape limits and frame-to-domain agreement, on the 256-band
// fixture where every limit is approached.
func TestFrameShapesAndAgreementAtMaximumWorkload(t *testing.T) {
	service := maximumWorkloadService(t)
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	world := service.world
	if len(frame.Bands) > domain.MaxBands {
		t.Fatalf("frame has %d bands", len(frame.Bands))
	}
	assignments, perBand, interbreed := 0, 0, 0
	for _, band := range frame.Bands {
		if len(band.MigrationCandidates) > 10 {
			t.Fatalf("band %d has %d migration candidates", band.ID, len(band.MigrationCandidates))
		}
		total := 0
		for _, share := range band.AllocationBP {
			total += int(share)
		}
		if total != 10_000 {
			t.Fatalf("band %d allocation totals %d basis points", band.ID, total)
		}
		for trait, value := range band.HeritableState {
			if value < 0 || value > 1 {
				t.Fatalf("band %d trait %d = %v outside [0, 1]", band.ID, trait, value)
			}
		}
		assignments += len(band.AllocationBP)
		perBand += len(band.MigrationCandidates) + len(band.ResearchProgress) + len(band.HeritableState) + len(band.PassageStatuses)
		seen := map[gameapi.BandID]bool{}
		for _, id := range band.InterbreedCandidateIDs {
			if seen[id] {
				t.Fatalf("band %d lists interbreed candidate %d twice", band.ID, id)
			}
			seen[id] = true
		}
		interbreed += len(band.InterbreedCandidateIDs)
		// The candidates must be exactly the co-located archaic bands.
		want := map[gameapi.BandID]bool{}
		if band.Species == gameapi.HomoSapiens {
			for _, other := range frame.Bands {
				if other.Species == gameapi.ArchaicHominin && other.TileID == band.TileID && other.Population > 0 {
					want[other.ID] = true
				}
			}
		}
		if !reflect.DeepEqual(seen, want) {
			t.Fatalf("band %d interbreed candidates %v, co-located archaic bands %v", band.ID, seen, want)
		}
	}
	if assignments > 5*len(frame.Bands) || assignments > 1_280 {
		t.Fatalf("assignment entries = %d", assignments)
	}
	if perBand > (10+9+6+3)*len(frame.Bands) || perBand > 7_168 {
		t.Fatalf("candidate, progress, heritable, and passage entries = %d", perBand)
	}
	if interbreed > 16_384 {
		t.Fatalf("interbreed candidate IDs = %d", interbreed)
	}
	grid := world.Grid()
	for id, tile := range frame.Tiles {
		geography, _ := grid.Tile(domain.TileID(id))
		if tile.ElevationKm != geography.ElevationKm {
			t.Fatalf("tile %d elevation %v, geography %v", id, tile.ElevationKm, geography.ElevationKm)
		}
		if tile.Explored != world.IsExplored(domain.TileID(id)) {
			t.Fatalf("tile %d explored %t, world %t", id, tile.Explored, world.IsExplored(domain.TileID(id)))
		}
		if !geography.Land {
			continue
		}
		// The summary must be the domain's regional profile, field for field.
		habitat := world.Habitat()[id]
		profile, ok := domain.FaunaFor(geography.Region, habitat.Biome, habitat.BaselineK > 0)
		if !ok {
			t.Fatalf("tile %d has no fauna profile", id)
		}
		want := gameapi.FaunaSummary{HuntingSupported: profile.HuntingSupported, MegafaunaSupported: profile.MegafaunaSupported}
		for group := domain.FaunaGroup(0); group < domain.FaunaGroupCount; group++ {
			want.Weights[mapFaunaGroup(group)] = profile.Weights[group]
		}
		if tile.Fauna != want {
			t.Fatalf("tile %d fauna %+v, domain profile %+v", id, tile.Fauna, want)
		}
	}
	// Each passage status is the domain predicate's verdict, coarsened: open
	// only when available, unavailable when the band is elsewhere or the far
	// side cannot hold it, locked for every spent-action, Beringia, or
	// navigation reason.
	for index, band := range world.Bands() {
		projected := frame.Bands[index]
		if projected.ID != gameapi.BandID(band.ID) {
			t.Fatalf("frame band %d is %d, world band is %d", index, projected.ID, band.ID)
		}
		for passage := domain.PassageID(0); passage < domain.PassageCount; passage++ {
			reason := world.PassageStatus(band.ID, passage)
			want := gameapi.PassageLocked
			switch reason {
			case domain.PassageAvailable:
				want = gameapi.PassageOpen
			case domain.PassageNotAtEndpoint, domain.PassageDestinationUninhabitable:
				want = gameapi.PassageUnavailable
			}
			if got := projected.PassageStatuses[passage]; got != want {
				t.Fatalf("band %d passage %d status %v, domain reason %v wants %v", band.ID, passage, got, reason, want)
			}
		}
		if projected.LastFoodReport.Turn != band.LastFoodReport.Turn || projected.LastFoodReport.RequiredFU != float64(band.LastFoodReport.RequiredFU) {
			t.Fatalf("band %d food report %+v, domain %+v", band.ID, projected.LastFoodReport, band.LastFoodReport)
		}
	}
}

// Step 6's passage-status agreement, branch by branch. The 256-band fixture has
// no band on a passage endpoint, so it only ever reaches "not at endpoint".
// Here a band is placed on each endpoint of each passage, with and without
// Coastal Navigation and a spent spatial action. The fixture's turn 300 has
// Beringia open; a new campaign's turn 0 has it closed. Every reason the
// domain predicate can give must turn up and project correctly.
func TestPassageStatusesMatchTheDomainPredicateForEveryReason(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/performance_profile_save.json")
	if err != nil {
		t.Fatal(err)
	}
	late, err := DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := NewGameService(1)
	early, err := fresh.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[domain.PassageAvailability]int{}
	beringia := map[bool]bool{}
	for _, base := range []SaveState{late, early} {
		climate, err := domain.ClimateAt(base.WorldSeed, base.Turn)
		if err != nil {
			t.Fatal(err)
		}
		beringia[domain.BeringiaOpen(climate.LongTermTempOffset)] = true
		coastal := uint16(1) << domain.CoastalNavigation
		withTech, withoutTech := -1, -1
		for index, band := range base.Bands {
			if band.AcquiredTech&coastal != 0 && withTech < 0 {
				withTech = index
			}
			if band.AcquiredTech&coastal == 0 && withoutTech < 0 {
				withoutTech = index
			}
		}
		for _, index := range []int{withTech, withoutTech} {
			if index < 0 {
				continue
			}
			for _, passage := range domain.Passages() {
				for _, endpoint := range []domain.TileID{passage.From, passage.To} {
					for _, spent := range []bool{false, true} {
						state := base
						state.Bands = append([]BandSave(nil), base.Bands...)
						state.Bands[index].TileID = uint16(endpoint)
						state.Bands[index].SpatialActionUsed = spent
						world, err := state.RestoreWorld()
						if err != nil {
							t.Fatalf("turn %d passage %d endpoint %d: %v", state.Turn, passage.ID, endpoint, err)
						}
						service, _ := NewGameService(1)
						service.world = world
						frame, err := service.Snapshot()
						if err != nil {
							t.Fatal(err)
						}
						for checked := domain.PassageID(0); checked < domain.PassageCount; checked++ {
							reason := world.PassageStatus(world.Bands()[index].ID, checked)
							reasons[reason]++
							want := gameapi.PassageLocked
							switch reason {
							case domain.PassageAvailable:
								want = gameapi.PassageOpen
							case domain.PassageNotAtEndpoint, domain.PassageDestinationUninhabitable:
								want = gameapi.PassageUnavailable
							}
							if got := frame.Bands[index].PassageStatuses[checked]; got != want {
								t.Fatalf("turn %d at endpoint %d of passage %d: passage %d status %v, domain reason %v wants %v",
									state.Turn, endpoint, passage.ID, checked, got, reason, want)
							}
						}
					}
				}
			}
		}
	}
	t.Logf("Beringia states %v, reasons %v", beringia, reasons)
	if len(beringia) != 2 {
		t.Errorf("fixtures cover Beringia states %v, want open and closed", beringia)
	}
	// Every endpoint's far side is habitable on all 400 turns of the
	// fixture's seed, so "destination uninhabitable" cannot be staged here.
	for reason := domain.PassageNotAtEndpoint; reason <= domain.PassageAvailable; reason++ {
		if reasons[reason] == 0 && reason != domain.PassageDestinationUninhabitable {
			t.Errorf("no placement produced domain reason %v", reason)
		}
	}
}
