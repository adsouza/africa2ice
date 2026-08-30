package domain

import (
	"fmt"
	"math"
)

type Population uint32
type Health float64
type FU float64
type WU float64
type Probability float64
type TraitValue float64
type AssignmentBP uint16

const MaxPopulation Population = 1<<32 - 1

func finiteNonNegative(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return fmt.Errorf("%w: %s", ErrInvalidValue, name)
	}
	return nil
}

// RoundPopulation is the domain's only audited float-to-population conversion.
// Its range checks make Go's truncating conversion deterministic; adding 0.5
// gives nearest-integer rounding with exact halves rounded upward.
func RoundPopulation(value float64) (Population, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(MaxPopulation) {
		return 0, fmt.Errorf("%w: population", ErrInvalidValue)
	}
	return Population(value + 0.5), nil
}
func ValidateFU(v FU) error { return finiteNonNegative("food units", float64(v)) }
func ValidateWU(v WU) error { return finiteNonNegative("water units", float64(v)) }

func ValidateUnitInterval(name string, v float64) error {
	if err := finiteNonNegative(name, v); err != nil {
		return err
	}
	if v > 1 {
		return fmt.Errorf("%w: %s exceeds one", ErrInvalidValue, name)
	}
	return nil
}

func ValidateHealth(v Health) error           { return ValidateUnitInterval("health", float64(v)) }
func ValidateProbability(v Probability) error { return ValidateUnitInterval("probability", float64(v)) }
func ValidateTraitValue(v TraitValue) error   { return ValidateUnitInterval("trait value", float64(v)) }

func ValidateAssignments(values [AssignmentCount]AssignmentBP) error {
	var total uint32
	for _, value := range values {
		total += uint32(value)
	}
	if total != 10_000 {
		return ErrInvalidAssignment
	}
	return nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
