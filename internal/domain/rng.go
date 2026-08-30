package domain

import rand "math/rand/v2"

const splitMixIncrement uint64 = 0x9e3779b97f4a7c15

func SplitMix64(state uint64) (uint64, uint64) {
	next := state + splitMixIncrement
	z := next
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return next, z ^ (z >> 31)
}

type WorldRNG struct{ pcg *rand.PCG }

func NewWorldRNG(seed uint64) *WorldRNG {
	state, word1 := SplitMix64(seed)
	_, word2 := SplitMix64(state)
	return &WorldRNG{pcg: rand.NewPCG(word1, word2)}
}

func (r *WorldRNG) Uint64() uint64 { return r.pcg.Uint64() }

func (r *WorldRNG) Float64() float64 {
	return float64(float64(r.pcg.Uint64()>>11) * 0x1p-53)
}

func (r *WorldRNG) MarshalBinary() ([]byte, error) { return r.pcg.MarshalBinary() }

func RestoreWorldRNG(state []byte) (*WorldRNG, error) {
	pcg := rand.NewPCG(0, 0)
	if err := pcg.UnmarshalBinary(state); err != nil {
		return nil, err
	}
	return &WorldRNG{pcg: pcg}, nil
}

func UnitNoiseV1(seed uint64, turn int) float64 {
	value := seed ^ 0x6a09e667f3bcc909 ^ uint64(turn)*0x9e3779b97f4a7c15
	_, mixed := SplitMix64(value)
	unit := float64(float64(mixed>>11) * 0x1p-53)
	return float64(2*unit) - 1
}

type ResourceDomain uint8

const (
	ResourceRegion ResourceDomain = iota
	ResourceTile
)

type ResourceStock uint8

const (
	FloraStock ResourceStock = iota
	FaunaStock
	WaterStock
)

func UnitResourceAbundanceV1(seed uint64, domain ResourceDomain, key uint64, stock ResourceStock) float64 {
	value := seed ^ 0xbb67ae8584caa73b
	value ^= uint64(domain+1) * 0x3c6ef372fe94f82b
	value ^= key * 0xa54ff53a5f1d36f1
	value ^= uint64(stock+1) * 0x510e527fade682d1
	_, mixed := SplitMix64(value)
	unit := float64(float64(mixed>>11) * 0x1p-53)
	return float64(2*unit) - 1
}
