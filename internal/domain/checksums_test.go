package domain

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"testing"
)

// DESIGN.md §12 steps 3 and 4 require this package to freeze independent
// checksums over the authored geography and the generated tables. They are
// independent on purpose: one combined hash tells you something moved, while
// separate hashes tell you *which* mask moved, which is the difference between
// a five-minute diagnosis and an afternoon.
//
// A checksum change is never a test to update. Every one of these values is an
// input to a saved campaign's reconstruction, so a change here reclassifies
// tiles in existing saves and requires a new algorithm version in
// AlgorithmVersions, per §5's admission discipline.

const (
	landMaskChecksum        = 0xec6d44777830a4ad
	waterMaskChecksum       = 0x58579657ea665905
	regionMaskChecksum      = 0x711cc3239db7432f
	highlandMaskChecksum    = 0x206e2e94b18cb711
	riverMaskChecksum       = 0xaed4591799a1b706
	coastalMaskChecksum     = 0x1f43bb03f1d038b9
	elevationChecksum       = 0x55e2e631a779654d
	naturalShelterChecksum  = 0x3d9a7aac56ee8ad9
	baseMoistureChecksum    = 0xc2893007e359e3c5
	latitudeTableChecksum   = 0xf15ec7fbe87b9a7b
	orbitalTableChecksum    = 0x31dccd8a94047833
	seasonalTableChecksum   = 0x18a2c4d6211e0ab8
	precessionTableChecksum = 0xd14491531e2ca7ce
)

// accumulator hashes exact bit patterns rather than formatted text, so a value
// that prints the same but differs in its last bit still moves the checksum.
type accumulator struct {
	hash    interface{ Write([]byte) (int, error) }
	scratch [8]byte
	sum     func() uint64
}

func newAccumulator() *accumulator {
	hash := fnv.New64a()
	return &accumulator{hash: hash, sum: hash.Sum64}
}

func (a *accumulator) addUint64(value uint64) {
	binary.BigEndian.PutUint64(a.scratch[:], value)
	_, _ = a.hash.Write(a.scratch[:])
}

func (a *accumulator) addFloat(value float64) { a.addUint64(math.Float64bits(value)) }

func (a *accumulator) addBool(value bool) {
	if value {
		a.addUint64(1)
		return
	}
	a.addUint64(0)
}

func (a *accumulator) addInt(value int) { a.addUint64(uint64(value)) }

func generatedGrid(t *testing.T) *Grid {
	t.Helper()
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	return grid
}

// tileChecksum walks all TileCount tiles in stable ID order.
func tileChecksum(t *testing.T, grid *Grid, add func(*accumulator, TileGeography)) uint64 {
	t.Helper()
	accumulated := newAccumulator()
	for id := range TileCount {
		geography, ok := grid.Tile(TileID(id))
		if !ok {
			t.Fatalf("tile %d is absent from a fully generated grid", id)
		}
		accumulated.addInt(id)
		add(accumulated, geography)
	}
	return accumulated.sum()
}

func tableChecksum(bits []uint64) uint64 {
	accumulated := newAccumulator()
	for index, value := range bits {
		accumulated.addInt(index)
		accumulated.addUint64(value)
	}
	return accumulated.sum()
}

func expectChecksum(t *testing.T, name string, got uint64, want uint64) {
	t.Helper()
	if got != want {
		t.Errorf("%s checksum = %#016x, frozen value is %#016x\n"+
			"    A change here reclassifies tiles in existing saves; it needs a new algorithm version, not a new constant.",
			name, got, want)
	}
}

func TestFrozenGeographyChecksums(t *testing.T) {
	grid := generatedGrid(t)

	expectChecksum(t, "land mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addBool(tile.Land)
	}), landMaskChecksum)

	// Water is asserted independently rather than as !Land, so that a future
	// third surface class cannot silently pass both masks.
	expectChecksum(t, "water mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addBool(!tile.Land)
	}), waterMaskChecksum)

	expectChecksum(t, "region mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addUint64(uint64(tile.Region))
	}), regionMaskChecksum)

	expectChecksum(t, "highland mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addBool(tile.ElevationKm >= HighlandElevationKm)
	}), highlandMaskChecksum)

	expectChecksum(t, "river mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addBool(tile.RiverCorridor)
	}), riverMaskChecksum)

	expectChecksum(t, "coastal mask", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addBool(tile.Coastal)
	}), coastalMaskChecksum)

	expectChecksum(t, "elevation", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addFloat(tile.ElevationKm)
	}), elevationChecksum)

	expectChecksum(t, "natural shelter", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addFloat(tile.NaturalShelter)
	}), naturalShelterChecksum)

	expectChecksum(t, "base moisture", tileChecksum(t, grid, func(a *accumulator, tile TileGeography) {
		a.addFloat(tile.BaseMoisture)
	}), baseMoistureChecksum)
}

func TestFrozenTableChecksums(t *testing.T) {
	expectChecksum(t, "latitude sin-squared", tableChecksum(latitudeSinSquaredBits[:]), latitudeTableChecksum)
	expectChecksum(t, "orbital sin", tableChecksum(orbitalSinBits[:]), orbitalTableChecksum)
	expectChecksum(t, "seasonal cos", tableChecksum(seasonalCosBits[:]), seasonalTableChecksum)
	expectChecksum(t, "precession sin", tableChecksum(precessionSinBits[:]), precessionTableChecksum)
}

// TestGeographyChecksumsAreSeedIndependent proves the authored geography is
// authored: no checksum above may move with the campaign seed.
func TestGeographyChecksumsAreSeedIndependent(t *testing.T) {
	reference := generatedGrid(t)
	want := tileChecksum(t, reference, func(a *accumulator, tile TileGeography) {
		a.addBool(tile.Land)
		a.addBool(tile.Coastal)
		a.addBool(tile.RiverCorridor)
		a.addUint64(uint64(tile.Region))
		a.addFloat(tile.ElevationKm)
		a.addFloat(tile.NaturalShelter)
		a.addFloat(tile.BaseMoisture)
	})
	for _, seed := range BalanceSeedCorpus {
		world, err := NewWorld(seed)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		got := tileChecksum(t, world.Grid(), func(a *accumulator, tile TileGeography) {
			a.addBool(tile.Land)
			a.addBool(tile.Coastal)
			a.addBool(tile.RiverCorridor)
			a.addUint64(uint64(tile.Region))
			a.addFloat(tile.ElevationKm)
			a.addFloat(tile.NaturalShelter)
			a.addFloat(tile.BaseMoisture)
		})
		if got != want {
			t.Fatalf("seed %d perturbed the authored geography: %#016x != %#016x", seed, got, want)
		}
	}
}
