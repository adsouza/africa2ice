package domain

const DiffusionRate = 0.20

type geneticPartner struct {
	index int
	rate  float64
}

func ordinaryContact(grid *Grid, left, right TileID) bool {
	if left == right {
		return true
	}
	for _, edge := range grid.OrdinaryEdges(left) {
		if edge.To == right {
			return true
		}
	}
	return false
}

func knowledgeContact(grid *Grid, left, right Band) bool {
	if left.Species != right.Species {
		return left.TileID == right.TileID
	}
	return ordinaryContact(grid, left.TileID, right.TileID)
}

func applyKnowledgeAndGenetics(bands []Band, grid *Grid, research map[BandID]float64, selection map[BandID]HeritableState, rng *WorldRNG) {
	snapshot := append([]Band(nil), bands...)
	geneticPartners := make([][]geneticPartner, len(snapshot))
	sourceCounts := make([][TechCount]int, len(snapshot))
	for left := 0; left < len(snapshot); left++ {
		for right := left + 1; right < len(snapshot); right++ {
			if knowledgeContact(grid, snapshot[left], snapshot[right]) {
				for technology := Technology(0); technology < TechCount; technology++ {
					if snapshot[right].Technology.Has(technology) {
						sourceCounts[left][technology]++
					}
					if snapshot[left].Technology.Has(technology) {
						sourceCounts[right][technology]++
					}
				}
			}
			if snapshot[left].Species == snapshot[right].Species && ordinaryContact(grid, snapshot[left].TileID, snapshot[right].TileID) {
				geneticPartners[left] = append(geneticPartners[left], geneticPartner{right, SameSpeciesGeneFlowRate})
				geneticPartners[right] = append(geneticPartners[right], geneticPartner{left, SameSpeciesGeneFlowRate})
			}
		}
	}
	for actorIndex, actor := range snapshot {
		if actor.Species != HomoSapiens || !actor.HasInterbreedTarget {
			continue
		}
		for targetIndex, target := range snapshot {
			if target.ID == actor.InterbreedTarget && target.Species == ArchaicHominin {
				geneticPartners[actorIndex] = append(geneticPartners[actorIndex], geneticPartner{targetIndex, InterbreedGeneFlowRate})
				geneticPartners[targetIndex] = append(geneticPartners[targetIndex], geneticPartner{actorIndex, InterbreedGeneFlowRate})
				break
			}
		}
	}

	for index := range bands {
		state := snapshot[index].Technology
		for technology := Technology(0); technology < TechCount; technology++ {
			if state.Has(technology) || !state.PrerequisitesMet(technology) {
				continue
			}
			gain := float64(float64(sourceCounts[index][technology]) * DiffusionRate * ResearchCost[technology])
			if state.HasTarget && state.Target == technology {
				gain += research[snapshot[index].ID]
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
		bands[index].Technology = state
	}

	for index := range bands {
		for trait := HeritableTrait(0); trait < HeritableTraitCount; trait++ {
			original := float64(snapshot[index].Heritable[trait])
			flow := geneFlowDelta(index, trait, snapshot, geneticPartners[index])
			mutation := 0.0
			if original == 0 && MutationProbability[trait] > 0 && rng.Float64() < MutationProbability[trait] {
				mutation = MutationEntryFrequency[trait]
			}
			delta := float64(selection[snapshot[index].ID][trait]) + flow + mutation
			bands[index].Heritable[trait] = TraitValue(clamp01(original + delta))
		}
	}
}

func geneFlowDelta(recipient int, trait HeritableTrait, snapshot []Band, partners []geneticPartner) float64 {
	if len(partners) == 0 {
		return 0
	}
	maximumWeight := 0.0
	for _, partner := range partners {
		weight := float64(partner.rate * float64(snapshot[partner.index].Population))
		if weight > maximumWeight {
			maximumWeight = weight
		}
	}
	if maximumWeight <= 0 {
		return 0
	}
	scaledInfluence, scaledTrait := 0.0, 0.0
	for _, partner := range partners {
		scaledWeight := float64(partner.rate*float64(snapshot[partner.index].Population)) / maximumWeight
		scaledInfluence += scaledWeight
		scaledTrait += float64(scaledWeight * float64(snapshot[partner.index].Heritable[trait]))
	}
	partnerMean := scaledTrait / scaledInfluence
	scale := maximumWeight
	recipientPopulation := float64(snapshot[recipient].Population)
	if recipientPopulation > scale {
		scale = recipientPopulation
	}
	influenceScaled := float64(scaledInfluence * (maximumWeight / scale))
	mix := influenceScaled / (recipientPopulation/scale + influenceScaled)
	if mix > MaxGeneFlowPerTurn {
		mix = MaxGeneFlowPerTurn
	}
	return float64(mix * (partnerMean - float64(snapshot[recipient].Heritable[trait])))
}
