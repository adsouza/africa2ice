package domain

import (
	"fmt"
	"math"
)

const (
	MaxResearchPerTurn     = 20.0
	ResearchHalfSaturation = 25.0
)

var ResearchCost = [TechCount]float64{80, 90, 70, 120, 100, 140, 130, 120, 150}
var technologyCapacityFactor = [TechCount]float64{1.05, 1.03, 1.04, 1.05, 1.04, 1.08, 1.06, 1.03, 1.02}
var technologyPrerequisites = [TechCount]uint16{
	0,
	0,
	0,
	1 << HaftedTools,
	1 << HaftedTools,
	1<<Firecraft | 1<<HaftedTools,
	1<<Firecraft | 1<<PlantKnowledge,
	1 << CordageAndNets,
	1 << CordageAndNets,
}

type TechnologyState struct {
	Acquired  uint16
	Target    Technology
	HasTarget bool
	Progress  [TechCount]float64
}

func (state TechnologyState) Validate() error {
	validMask := uint16(1<<TechCount) - 1
	if state.Acquired&^validMask != 0 {
		return fmt.Errorf("%w: technology bitset", ErrInvalidValue)
	}
	for technology := Technology(0); technology < TechCount; technology++ {
		progress := state.Progress[technology]
		if math.IsNaN(progress) || math.IsInf(progress, 0) || progress < 0 || progress > ResearchCost[technology] {
			return fmt.Errorf("%w: research progress", ErrInvalidValue)
		}
		if state.Has(technology) {
			if !state.PrerequisitesMet(technology) || progress != ResearchCost[technology] {
				return fmt.Errorf("%w: acquired technology", ErrInvalidValue)
			}
		}
	}
	if state.HasTarget && (state.Target >= TechCount || state.Has(state.Target) || !state.PrerequisitesMet(state.Target)) {
		return fmt.Errorf("%w: research target", ErrInvalidValue)
	}
	return nil
}

func (state TechnologyState) Has(technology Technology) bool {
	return technology < TechCount && state.Acquired&(1<<technology) != 0
}

func (state TechnologyState) PrerequisitesMet(technology Technology) bool {
	if technology >= TechCount {
		return false
	}
	required := technologyPrerequisites[technology]
	return state.Acquired&required == required
}

func (state *TechnologyState) Select(technology Technology) error {
	if technology >= TechCount {
		return ErrInvalidValue
	}
	if state.Has(technology) {
		return ErrTechnologyAlreadyAcquired
	}
	if !state.PrerequisitesMet(technology) {
		return ErrMissingTechnologyPrerequisite
	}
	state.Target, state.HasTarget = technology, true
	return nil
}

func ResearchGain(workers float64) float64 {
	if workers <= 0 {
		return 0
	}
	if workers >= ResearchHalfSaturation {
		q := 1 / (1 + ResearchHalfSaturation/workers)
		return float64(MaxResearchPerTurn * q)
	}
	q := workers / (workers + ResearchHalfSaturation)
	return float64(MaxResearchPerTurn * q)
}

// PlannedResearchGain is what one turn of toolcraft adds to the band's current
// research target: the production curve, capped by what the target still needs.
// It returns zero when there is nothing to research.
//
// The turn pipeline computes this during resolution but applies it later, in
// applyKnowledgeAndGenetics, where it lands alongside whatever the same
// technology gained from contact with neighbouring bands.
func (state TechnologyState) PlannedResearchGain(workers float64) float64 {
	if !state.HasTarget || state.Target >= TechCount || state.Has(state.Target) || !state.PrerequisitesMet(state.Target) {
		return 0
	}
	remaining := ResearchCost[state.Target] - state.Progress[state.Target]
	gain := ResearchGain(workers)
	if gain > remaining {
		gain = remaining
	}
	return gain
}

// AdvanceResearch adds gain to one technology's progress, acquiring it and
// clearing the research target once the cost is met. It is the only place
// progress becomes ownership, so research and knowledge diffusion complete a
// technology by the same rule.
func (state *TechnologyState) AdvanceResearch(technology Technology, gain float64) {
	if technology >= TechCount || state.Has(technology) || !state.PrerequisitesMet(technology) {
		return
	}
	progress := state.Progress[technology] + gain
	if progress >= ResearchCost[technology] {
		progress = ResearchCost[technology]
		state.Acquired |= 1 << technology
		if state.HasTarget && state.Target == technology {
			state.HasTarget = false
		}
	}
	state.Progress[technology] = progress
}

func (state TechnologyState) CapacityMultiplier() float64 {
	result := 1.0
	for technology := Technology(0); technology < TechCount; technology++ {
		if state.Has(technology) {
			result = float64(result * technologyCapacityFactor[technology])
		}
	}
	return result
}
