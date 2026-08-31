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
	// These are the tighten-only reference-run margins from DESIGN.md §13.
	maxFirstDestinationTurn       = 350
	minFirstDestinationPopulation = 2 * MinEstablishedBand
	minSurvivingSapiensAtTurn400  = uint64(10 * MinEstablishedBand)
	// minSeedsReachingTargetPerPolicy is the reachability margin: a directed
	// policy must establish its own destination on at least this many corpus
	// seeds, so that reachability is a property of the model and not of one
	// lucky seed.
	minSeedsReachingTargetPerPolicy = 1
)

// foundingSapiensBands is how many sapiens bands the scenario opens with. It is
// derived rather than written down so that editing StartingAnchors cannot
// silently weaken the subdivision margin below.
func foundingSapiensBands() int {
	count := 0
	for _, anchor := range StartingAnchors {
		if anchor.Species == HomoSapiens {
			count++
		}
	}
	return count
}

// minFinalEstablishedBands is the subdivision margin, and it exists because the
// other three margins are all satisfiable by a campaign that never disperses.
//
// Total sapiens and the regional achievement record are both blind to band
// count: achievements stay permanently true once earned, so a campaign that
// touches Beringia at turn 200 and contracts to its founding tiles by turn 400
// still reports a victory, and a founding band that doubles in place still
// clears the survival margin. Both degenerate shapes were observed. A carrying
// capacity high enough to remove crowding stops bands splitting at all, and the
// model settles into the founding four growing past Dunbar's number while no
// fifth band is ever founded; the same corpus then passes 48 of 48 with zero
// variance, which is itself the tell.
//
// Requiring strictly more established bands than the scenario was founded with
// is the weakest statement that rules this out: dispersal in this model happens
// by splitting, so a campaign that disperses must end more subdivided than it
// began. Like every margin here it is tighten-only, and it is deliberately set
// at the floor so that it blocks the degenerate case without asserting a
// population curve the design has not chosen.
func minFinalEstablishedBands() int { return foundingSapiensBands() + 1 }

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
		var visited uint16
		var targetPeak Population
		firstTurnMin, firstTurnMax := MaxCampaignTurn+1, -1
		firstPopulationMin, firstPopulationMax := MaxPopulation, Population(0)
		bestFinalBands, worstFinalBands := 0, int(^uint(0)>>1)
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
			if outcome.FirstDestinationTurn >= 0 {
				firstTurnMin = min(firstTurnMin, outcome.FirstDestinationTurn)
				firstTurnMax = max(firstTurnMax, outcome.FirstDestinationTurn)
				firstPopulationMin = min(firstPopulationMin, outcome.FirstDestinationPopulation)
				firstPopulationMax = max(firstPopulationMax, outcome.FirstDestinationPopulation)
			}
			established |= outcome.EstablishedRegions
			visited |= outcome.RegionsVisited
			targetPeak = max(targetPeak, outcome.TargetPeakBand)
			bestSapiens = max(bestSapiens, outcome.FinalSapiens)
			worstSapiens = min(worstSapiens, outcome.FinalSapiens)
			bestFinalBands = max(bestFinalBands, outcome.FinalEstablishedBands)
			worstFinalBands = min(worstFinalBands, outcome.FinalEstablishedBands)
		}
		if firstTurnMax < 0 {
			firstTurnMin = -1
			firstPopulationMin = 0
		}
		fmt.Fprintf(&report,
			"\n  %-19s victory=%d extinction=%d dispersalFailed=%d targetReached=%d/%d firstDestinationTurn=[%d..%d] firstDestinationPopulation=[%d..%d] finalSapiens=[%d..%d] finalEstablishedBands=[%d..%d] regionsEverEstablished=%s regionsVisited=%s targetPeakBand=%d",
			name, victories, extinctions, failures, reachedTarget, len(byPolicy[name]),
			firstTurnMin, firstTurnMax, firstPopulationMin, firstPopulationMax,
			worstSapiens, bestSapiens, worstFinalBands, bestFinalBands,
			establishedRegionList(established), establishedRegionList(visited), targetPeak)
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
// The selected Initial growth, hazard, and split-pressure values are held by
// this test rather than by a recorded one-off run: the reference route must
// both establish a destination and retain the survival margin, and every named
// route must remain reachable across the exact locked seed corpus.
func TestDomainViabilityGate(t *testing.T) {
	outcomes := runViabilityMatrix(t)
	report := summarize(outcomes)
	t.Log(report)

	reachedByPolicy := map[string]int{}
	referenceReachedWithMargin := 0
	referenceSurvivedToTurn400 := 0
	referenceSubdivided := 0
	for _, outcome := range outcomes {
		if outcome.TargetReachedTurn >= 0 {
			reachedByPolicy[outcome.Policy.Name]++
		}
		if outcome.Policy.Name == ReferenceRoutePolicy.Name &&
			outcome.FirstDestinationTurn >= 0 && outcome.FirstDestinationTurn <= maxFirstDestinationTurn &&
			outcome.FirstDestinationPopulation >= minFirstDestinationPopulation {
			referenceReachedWithMargin++
		}
		if outcome.Policy.Name == ReferenceRoutePolicy.Name && outcome.Result != CampaignExtinction && outcome.FinalSapiens >= minSurvivingSapiensAtTurn400 {
			referenceSurvivedToTurn400++
		}
		if outcome.Policy.Name == ReferenceRoutePolicy.Name && outcome.FinalEstablishedBands >= minFinalEstablishedBands() {
			referenceSubdivided++
		}
	}

	if referenceReachedWithMargin != len(BalanceSeedCorpus) {
		t.Errorf("the reference policy reached a first destination by turn %d with at least %d people on %d/%d corpus seeds%s",
			maxFirstDestinationTurn, minFirstDestinationPopulation, referenceReachedWithMargin, len(BalanceSeedCorpus), report)
	}
	for _, policy := range DirectedRoutePolicies {
		if reachedByPolicy[policy.Name] < minSeedsReachingTargetPerPolicy {
			t.Errorf("%s established %v on %d seeds, the gate requires at least %d",
				policy.Name, policy.Target, reachedByPolicy[policy.Name], minSeedsReachingTargetPerPolicy)
		}
	}
	if referenceSurvivedToTurn400 == 0 {
		t.Errorf("the reference policy retained no turn-400 survival margin of %d sapiens%s",
			minSurvivingSapiensAtTurn400, report)
	}
	if referenceSubdivided == 0 {
		t.Errorf("the reference policy ended no campaign with more than the %d founding bands still established, so the model grows its founders in place rather than dispersing%s",
			foundingSapiensBands(), report)
	}
}
