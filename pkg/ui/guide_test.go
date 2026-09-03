package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestGuideAdvancesOnDonePredicatesAndNextButOnlyDismissEnds(t *testing.T) {
	guide := NewGuideState(false)
	if !guide.Visible() || guide.Step != GuideMove {
		t.Fatalf("fresh guide = %+v", guide)
	}
	band := &gameapi.Band{Species: gameapi.HomoSapiens}
	if guide = guide.Observe(band); guide.Step != GuideMove {
		t.Fatal("guide advanced without a move")
	}
	band.HasQueuedMigration = true
	if guide = guide.Observe(band); guide.Step != GuideResearch {
		t.Fatalf("after move step = %v, want Research", guide.Step)
	}
	// Research not chosen yet: stays; Next skips it explicitly.
	if guide = guide.Observe(band); guide.Step != GuideResearch {
		t.Fatal("research step advanced without a target")
	}
	guide = guide.Next()
	if guide.Step != GuideWorkforce {
		t.Fatalf("after Next = %v, want Workforce", guide.Step)
	}
	guide = guide.Next()
	if guide.Step != GuideEndTurn {
		t.Fatalf("after second Next = %v, want EndTurn", guide.Step)
	}
	if guide = guide.ObserveTurnCompleted(); guide.Step != GuideClosing {
		t.Fatalf("after turn = %v, want Closing", guide.Step)
	}
	if guide = guide.Next(); guide.Step != GuideClosing || !guide.Visible() {
		t.Fatal("Next left the closing card; only the × may")
	}
	if guide = guide.ObserveTurnCompleted(); guide.Step != GuideClosing {
		t.Fatal("a later turn changed the closing card")
	}
	if guide = guide.Dismiss(); guide.Visible() || guide.Step != GuideDismissed {
		t.Fatalf("dismissed = %+v", guide)
	}
	if guide = guide.Observe(band).Next().ObserveTurnCompleted(); guide.Step != GuideDismissed {
		t.Fatal("dismissed guide came back")
	}
	if NewGuideState(true).Visible() {
		t.Fatal("persisted dismissal was ignored")
	}
}

// TestGuideSkipsAlreadySatisfiedSteps covers the reviewer-found defect:
// Observe advanced at most one step, so a player who chose Research before
// Move left the guide on Move; completing Move then advanced only to
// Research even though it was already satisfied too.
func TestGuideSkipsAlreadySatisfiedSteps(t *testing.T) {
	guide := GuideState{Step: GuideMove}
	band := &gameapi.Band{Species: gameapi.HomoSapiens, HasQueuedMigration: true, HasResearchTarget: true}
	if guide = guide.Observe(band); guide.Step != GuideWorkforce {
		t.Fatalf("guide landed on %v after one Observe, want GuideWorkforce", guide.Step)
	}
}

func TestGuideCopyAndProgress(t *testing.T) {
	steps := []GuideStep{GuideMove, GuideResearch, GuideWorkforce, GuideEndTurn}
	for index, step := range steps {
		guide := GuideState{Step: step}
		current, total := guide.Progress()
		if current != index+1 || total != 4 {
			t.Fatalf("progress for %v = %d/%d", step, current, total)
		}
		if guide.Title() == "" || guide.Body() == "" {
			t.Fatalf("step %v has empty copy", step)
		}
	}
	closing := GuideState{Step: GuideClosing}
	if current, total := closing.Progress(); current != 4 || total != 4 {
		t.Fatalf("closing progress = %d/%d", current, total)
	}
	if !strings.Contains(closing.Body(), "Tab") || !strings.Contains(closing.Body(), "×") {
		t.Fatalf("closing copy = %q, want Tab and × mentioned", closing.Body())
	}
	if !strings.Contains(GuideState{Step: GuideMove}.Body(), "gold") {
		t.Fatal("move step copy does not mention the gold outline")
	}
	moveBody := GuideState{Step: GuideMove}.Body()
	if !strings.Contains(moveBody, "Best tile") {
		t.Fatal("move step copy does not name the Best tile button")
	}
	if strings.Contains(moveBody, "Move to gold tile") {
		t.Fatal("move step copy names a control that does not exist")
	}
}
