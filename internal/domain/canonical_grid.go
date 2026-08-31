package domain

import (
	"fmt"
	"sync"
)

var (
	canonicalGridOnce  sync.Once
	canonicalGridValue *Grid
	canonicalGridErr   error
)

// canonicalGrid returns the immutable, seed-independent geography and biome
// history shared by every world in this process. Per-seed habitat, resources,
// bands, exploration, and RNG state remain world-owned.
func canonicalGrid() (*Grid, error) {
	canonicalGridOnce.Do(func() {
		canonicalGridValue, canonicalGridErr = (WorldGenerator{}).Generate()
		if canonicalGridErr != nil {
			return
		}
		if err := ValidatePassages(canonicalGridValue); err != nil {
			canonicalGridValue = nil
			canonicalGridErr = fmt.Errorf("%w: passage catalog", err)
		}
	})
	return canonicalGridValue, canonicalGridErr
}
