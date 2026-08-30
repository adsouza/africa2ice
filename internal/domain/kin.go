package domain

// Kin support. A band in ordinary contact with same-species bands — sharing or
// neighbouring its tile — faces less acute risk than one standing alone. This
// is an Allee effect: mutual aid, shared watch, and shared knowledge of the
// ground make a shock survivable that would otherwise take a chunk of the band.
//
// It reuses ordinaryContact, the same predicate same-species gene flow already
// uses, so the model has one notion of a neighbouring band rather than two that
// can drift apart.
//
// It attaches to acute risk rather than to fertility because acute events are
// the only term that moves an integer population under the current calibration:
// the logistic growth term is smaller than RoundPopulation's half-person
// threshold in every band-turn measured, so a fertility bonus has nothing to
// act on. Revisit that once population growth itself can register.

const (
	KinContactHalfSaturation = 1.0
	MaxKinAcuteReduction     = 0.40
)

// kinContactShare is the saturating weight in [0, 1) a band derives from its
// living same-species neighbours. It saturates rather than accumulating so a
// dense cluster cannot outrun the cap.
func kinContactShare(contacts int) float64 {
	if contacts <= 0 {
		return 0
	}
	return float64(contacts) / (float64(contacts) + KinContactHalfSaturation)
}

// KinSupportRemainingRisk is the fraction of acute risk that survives kin
// support, following the same remaining-risk shape as InnateImmuneRemainingRisk
// and AridHeatRemaining. It is exactly 1 for an isolated band, so the rule is
// invisible where it does not apply, and never falls below its stated floor.
func KinSupportRemainingRisk(contacts int) float64 {
	return 1 - float64(MaxKinAcuteReduction*kinContactShare(contacts))
}

// kinContactCounts reports, for each band in the snapshot, how many living
// same-species bands share or neighbour its tile. It reads a snapshot rather
// than live state so the count cannot depend on the order the turn pipeline
// happens to walk the bands in.
func kinContactCounts(snapshot []Band, grid *Grid) []int {
	counts := make([]int, len(snapshot))
	for left := range snapshot {
		if snapshot[left].Population == 0 {
			continue
		}
		for right := left + 1; right < len(snapshot); right++ {
			if snapshot[right].Population == 0 || snapshot[left].Species != snapshot[right].Species {
				continue
			}
			if ordinaryContact(grid, snapshot[left].TileID, snapshot[right].TileID) {
				counts[left]++
				counts[right]++
			}
		}
	}
	return counts
}
