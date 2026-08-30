package domain

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// The step-5e mandatory domain viability gate. DESIGN.md §12 places it before
// step 6 so that a nonviable model is corrected before wire, storage, and UI
// contracts make correcting it expensive.
//
// Extinction and dispersal failure are recorded here as outcomes, never as
// harness errors: a campaign that ends badly is a result the model is allowed to
// produce. What the gate asserts is that the model can *also* produce the
// campaign the design is about.

// The margins this gate holds the model to. They are stated as named constants
// rather than inlined so that a future tuning pass changes a documented number
// and not the shape of the assertion.
const (
	// minSurvivingSapiensAtTurn400 is the turn-400 survival margin: a campaign
	// that reaches the end must still hold at least one establishable band.
	minSurvivingSapiensAtTurn400 = uint64(MinEstablishedBand)
	// minSeedsReachingTargetPerPolicy is the reachability margin: a directed
	// policy must establish its own destination on at least this many corpus
	// seeds, so that reachability is a property of the model and not of one
	// lucky seed.
	minSeedsReachingTargetPerPolicy = 1
)

func allRoutePolicies() []RoutePolicy {
	return append([]RoutePolicy{ReferenceRoutePolicy}, DirectedRoutePolicies[:]...)
}

// runViabilityMatrix plays every (corpus seed, policy) campaign once. The
// result is memoised because two tests consume it and a full matrix is 48
// four-hundred-turn campaigns; the harness determinism test above is what earns
// the right to reuse it.
var (
	viabilityOnce   sync.Once
	viabilityMatrix []CampaignOutcome
	viabilityErr    error
)

func runViabilityMatrix(t *testing.T) []CampaignOutcome {
	t.Helper()
	viabilityOnce.Do(func() {
		for _, policy := range allRoutePolicies() {
			for _, seed := range BalanceSeedCorpus {
				outcome, err := RunPolicyCampaign(seed, policy)
				if err != nil {
					// Only a harness or invariant failure reaches here. A lost
					// campaign returns an outcome, not an error.
					viabilityErr = fmt.Errorf("%s/%#x: %w", policy.Name, seed, err)
					return
				}
				viabilityMatrix = append(viabilityMatrix, outcome)
			}
		}
	})
	if viabilityErr != nil {
		t.Fatal(viabilityErr)
	}
	return viabilityMatrix
}

func summarize(outcomes []CampaignOutcome) string {
	var report strings.Builder
	byPolicy := map[string][]CampaignOutcome{}
	order := make([]string, 0, len(allRoutePolicies()))
	for _, outcome := range outcomes {
		if _, seen := byPolicy[outcome.Policy.Name]; !seen {
			order = append(order, outcome.Policy.Name)
		}
		byPolicy[outcome.Policy.Name] = append(byPolicy[outcome.Policy.Name], outcome)
	}
	for _, name := range order {
		var victories, extinctions, failures, reachedTarget int
		var bestSapiens, worstSapiens uint64
		worstSapiens = ^uint64(0)
		established := uint16(0)
		for _, outcome := range byPolicy[name] {
			switch outcome.Result {
			case CampaignVictory:
				victories++
			case CampaignExtinction:
				extinctions++
			default:
				failures++
			}
			if outcome.TargetReachedTurn >= 0 {
				reachedTarget++
			}
			established |= outcome.EstablishedRegions
			bestSapiens = max(bestSapiens, outcome.FinalSapiens)
			worstSapiens = min(worstSapiens, outcome.FinalSapiens)
		}
		fmt.Fprintf(&report,
			"\n  %-19s victory=%d extinction=%d dispersalFailed=%d targetReached=%d/%d finalSapiens=[%d..%d] regionsEverEstablished=%s",
			name, victories, extinctions, failures, reachedTarget, len(byPolicy[name]),
			worstSapiens, bestSapiens, establishedRegionList(established))
	}
	return report.String()
}

func establishedRegionList(mask uint16) string {
	names := make([]string, 0, RegionCount)
	for region := Region(0); region < RegionCount; region++ {
		if mask&(1<<region) != 0 {
			names = append(names, fmt.Sprintf("%v", region))
		}
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ",")
}

// TestViabilityHarnessIsDeterministic proves a campaign is a pure function of
// (seed, policy). Without this, every other assertion in this file would be
// measuring noise.
func TestViabilityHarnessIsDeterministic(t *testing.T) {
	for _, policy := range []RoutePolicy{ReferenceRoutePolicy, DirectedRoutePolicies[0]} {
		seed := BalanceSeedCorpus[0]
		first, err := RunPolicyCampaign(seed, policy)
		if err != nil {
			t.Fatal(err)
		}
		second, err := RunPolicyCampaign(seed, policy)
		if err != nil {
			t.Fatal(err)
		}
		if first != second {
			t.Fatalf("%s is not deterministic:\n  %+v\n  %+v", policy.Name, first, second)
		}
	}
}

// TestCampaignsRespectStructuralBounds covers the invariants that hold whether
// or not the model disperses, so a balance change cannot quietly break them.
func TestCampaignsRespectStructuralBounds(t *testing.T) {
	for _, outcome := range runViabilityMatrix(t) {
		if outcome.PeakBands > MaxBands {
			t.Errorf("%s/%#x: peaked at %d bands, cap is %d", outcome.Policy.Name, outcome.Seed, outcome.PeakBands, MaxBands)
		}
		if outcome.Result == CampaignOngoing {
			t.Errorf("%s/%#x: campaign ended still ongoing at turn %d", outcome.Policy.Name, outcome.Seed, outcome.TurnsCompleted)
		}
		if outcome.Result != CampaignExtinction && outcome.TurnsCompleted != MaxCampaignTurn {
			t.Errorf("%s/%#x: surviving campaign stopped at turn %d, want %d",
				outcome.Policy.Name, outcome.Seed, outcome.TurnsCompleted, MaxCampaignTurn)
		}
		if outcome.Result == CampaignExtinction && outcome.FinalSapiens != 0 {
			t.Errorf("%s/%#x: extinction recorded with %d sapiens alive", outcome.Policy.Name, outcome.Seed, outcome.FinalSapiens)
		}
	}
}

// TestDomainViabilityGate is the gate itself: can this model produce the
// campaign the design describes?
//
// STATUS: this gate currently FAILS, and the failure is a finding about the
// balance of the domain model rather than a defect in the harness above.
// Across all 48 (seed, policy) campaigns no destination region is ever
// established. The corridor is not the constraint — Arabia holds 10-43
// habitable tiles and the Levant 22-28 at every sampled turn — the demographics
// are: total sapiens population falls from a peak of ~400 to double digits, so
// no band ever carries the surplus that establishing a distant region needs.
//
// §12 step 5e is explicit that this pass may tune Appendix C **Initial** domain
// values, and must update their owning §7 rules, Appendix B, fixtures, and
// manifest rows together. It may not relax a tighten-only margin or change a
// **Locked** value to manufacture a win. Closing this gate is therefore a
// balance decision, not a code change.
func TestDomainViabilityGate(t *testing.T) {
	outcomes := runViabilityMatrix(t)
	report := summarize(outcomes)

	reachedByPolicy := map[string]int{}
	referenceReachedADestination := 0
	survivedToTurn400 := 0
	for _, outcome := range outcomes {
		if outcome.TargetReachedTurn >= 0 {
			reachedByPolicy[outcome.Policy.Name]++
		}
		if outcome.Policy.Name == ReferenceRoutePolicy.Name && outcome.FirstDestinationTurn >= 0 {
			referenceReachedADestination++
		}
		if outcome.Result != CampaignExtinction && outcome.FinalSapiens >= minSurvivingSapiensAtTurn400 {
			survivedToTurn400++
		}
	}

	if referenceReachedADestination == 0 {
		t.Errorf("the reference policy reached no destination region on any of the %d corpus seeds%s",
			len(BalanceSeedCorpus), report)
	}
	for _, policy := range DirectedRoutePolicies {
		if reachedByPolicy[policy.Name] < minSeedsReachingTargetPerPolicy {
			t.Errorf("%s established %v on %d seeds, the gate requires at least %d",
				policy.Name, policy.Target, reachedByPolicy[policy.Name], minSeedsReachingTargetPerPolicy)
		}
	}
	if survivedToTurn400 == 0 {
		t.Errorf("no campaign retained the turn-400 survival margin of %d sapiens", minSurvivingSapiensAtTurn400)
	}
}
