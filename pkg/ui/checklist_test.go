package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func checklistFrame() *gameapi.Frame {
	return &gameapi.Frame{
		CampaignResult: gameapi.Ongoing,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 5, Y: 5, Land: true, Explored: true, Biome: gameapi.Savanna},
			{ID: 1, X: 6, Y: 4, Land: true, Explored: true, Biome: gameapi.RiverineWoodland},
		},
		Bands: []gameapi.Band{
			{ID: 7, Species: gameapi.HomoSapiens, Population: 60, TileID: 0},
			{ID: 8, Species: gameapi.HomoSapiens, Population: 60, TileID: 0},
		},
	}
}

func TestDefaultOpenRowIsFirstUnfinishedRow(t *testing.T) {
	band := gameapi.Band{Species: gameapi.HomoSapiens}
	if got := DefaultOpenRow(&band); got != RowMove {
		t.Fatalf("fresh band opens %v, want Move", got)
	}
	band.HasQueuedMigration = true
	if got := DefaultOpenRow(&band); got != RowResearch {
		t.Fatalf("moved band opens %v, want Research", got)
	}
	band.HasResearchTarget = true
	if got := DefaultOpenRow(&band); got != RowMove {
		t.Fatalf("finished band opens %v, want Move fallback", got)
	}
	if got := DefaultOpenRow(nil); got != RowMove {
		t.Fatalf("nil band opens %v, want Move", got)
	}
}

func TestSummariesNameTheAcceptedState(t *testing.T) {
	frame := checklistFrame()
	band := frame.Bands[0]
	if got := MoveSummary(frame, band); got != "Choose a destination" {
		t.Fatalf("idle move summary = %q", got)
	}
	band.HasQueuedMigration, band.QueuedMigration = true, 1
	if got := MoveSummary(frame, band); got != "Move set → Riverine Woodland, NE" {
		t.Fatalf("queued move summary = %q", got)
	}
	band = frame.Bands[0]
	band.HasInterbreedTarget, band.InterbreedTargetID = true, 12
	if got := MoveSummary(frame, band); got != "Interbreeding with B12" {
		t.Fatalf("interbreed summary = %q", got)
	}
	band = frame.Bands[0]
	band.SpatialActionUsed = true
	if got := MoveSummary(frame, band); got != "Split queued" {
		t.Fatalf("split summary = %q", got)
	}

	research := gameapi.Band{}
	if got := ResearchSummary(research); got != "No target · choose one" {
		t.Fatalf("idle research summary = %q", got)
	}
	research.HasResearchTarget, research.ResearchTarget = true, gameapi.Firecraft
	research.ResearchProgress[gameapi.Firecraft] = 66
	research.ResearchOptions[gameapi.Firecraft].Cost = 80
	research.OriginalResearchGainPreview = 5.8
	if got := ResearchSummary(research); got != "Firecraft 66/80 · +5.8/turn" {
		t.Fatalf("research summary = %q", got)
	}

	allocation := [gameapi.AssignmentCount]uint16{3_500, 3_000, 1_500, 500, 1_500}
	if got := WorkforceSummary(allocation, false); got != "F 35 · H 30 · T 15 · M 5 · S 15" {
		t.Fatalf("workforce summary = %q", got)
	}
	if got := WorkforceSummary(allocation, true); got != "Unapplied changes" {
		t.Fatalf("dirty workforce summary = %q", got)
	}
}

func TestEndTurnGateOrdersHardBlocksBeforeTheSoftBlock(t *testing.T) {
	frame := checklistFrame()
	if gate := EndTurnGateFor(frame, true, true, false); gate.Enabled || gate.Label != "End turn · apply or discard workforce changes" {
		t.Fatalf("dirty draft gate = %+v", gate)
	}
	if gate := EndTurnGateFor(frame, false, true, false); gate.Enabled || gate.Label != "End turn · queue or clear the arrow-key choice" {
		t.Fatalf("pending cursor gate = %+v", gate)
	}
	soft := EndTurnGateFor(frame, false, false, false)
	if !soft.Enabled || !soft.Soft || soft.Label != "End turn · 2 bands still need a move" {
		t.Fatalf("soft gate = %+v", soft)
	}
	armed := EndTurnGateFor(frame, false, false, true)
	if !armed.Enabled || armed.Soft || armed.Label != "End turn now · Space" {
		t.Fatalf("armed gate = %+v", armed)
	}
	frame.Bands[0].SpatialActionUsed = true
	frame.Bands[1].HasQueuedMigration = true
	clear := EndTurnGateFor(frame, false, false, false)
	if !clear.Enabled || clear.Soft || clear.Label != "End turn · Space" {
		t.Fatalf("clear gate = %+v", clear)
	}
	one := checklistFrame()
	one.Bands[1].SpatialActionUsed = true
	if gate := EndTurnGateFor(one, false, false, false); gate.Label != "End turn · 1 band still needs a move" {
		t.Fatalf("singular gate = %+v", gate)
	}
}

func TestCompassDirectionNamesEightNeighbours(t *testing.T) {
	frame := &gameapi.Frame{Tiles: []gameapi.Tile{{ID: 0, X: 5, Y: 5}, {ID: 1, X: 6, Y: 4}, {ID: 2, X: 5, Y: 6}, {ID: 3, X: 9, Y: 9}}}
	if got := CompassDirection(frame, 0, 1); got != "NE" {
		t.Fatalf("NE = %q", got)
	}
	if got := CompassDirection(frame, 0, 2); got != "S" {
		t.Fatalf("S = %q", got)
	}
	if got := CompassDirection(frame, 0, 3); got != "" {
		t.Fatalf("distant = %q, want empty", got)
	}
}
