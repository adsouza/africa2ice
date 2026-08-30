package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestVisibleSapiensBandWindowPagesToTheSelection(t *testing.T) {
	bands := interleavedBandWindowFixture(12)
	tests := []struct {
		name        string
		selected    gameapi.BandID
		wantFirst   int
		wantLabel   string
		wantBandIDs []gameapi.BandID
	}{
		{name: "first page", selected: 1, wantFirst: 0, wantLabel: "Sapiens bands 1–5/12", wantBandIDs: []gameapi.BandID{1, 2, 3, 4, 5}},
		{name: "second page", selected: 6, wantFirst: 5, wantLabel: "Sapiens bands 6–10/12", wantBandIDs: []gameapi.BandID{6, 7, 8, 9, 10}},
		{name: "partial final page", selected: 12, wantFirst: 10, wantLabel: "Sapiens bands 11–12/12", wantBandIDs: []gameapi.BandID{11, 12}},
		{name: "missing selection uses first page", selected: 999, wantFirst: 0, wantLabel: "Sapiens bands 1–5/12", wantBandIDs: []gameapi.BandID{1, 2, 3, 4, 5}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			window := visibleSapiensBandWindow(bands, test.selected)
			if window.total != 12 || window.first != test.wantFirst || window.count != len(test.wantBandIDs) || window.label() != test.wantLabel {
				t.Fatalf("window = %+v", window)
			}
			for row, wantID := range test.wantBandIDs {
				band := bands[window.indices[row]]
				if band.Species != gameapi.HomoSapiens || band.ID != wantID {
					t.Fatalf("row %d = %#v, want sapiens band %d", row, band, wantID)
				}
			}
		})
	}
}

func TestVisibleSapiensBandWindowDoesNotPageSmallLists(t *testing.T) {
	bands := interleavedBandWindowFixture(4)
	window := visibleSapiensBandWindow(bands, 4)
	if window.total != 4 || window.first != 0 || window.count != 4 || window.label() != "Sapiens bands: 4" {
		t.Fatalf("window = %+v", window)
	}
}

func TestMaximumBandPageFitsBeforeOutcomeDetails(t *testing.T) {
	if bandListOriginY+maxVisibleSapiensBandRows*bandRowHeight > bandOutcomeOriginY {
		t.Fatal("band page overlaps the outcome details")
	}
}

func interleavedBandWindowFixture(sapiensCount int) []gameapi.Band {
	bands := make([]gameapi.Band, 0, sapiensCount*2)
	for id := 1; id <= sapiensCount; id++ {
		bands = append(bands,
			gameapi.Band{ID: gameapi.BandID(10_000 + id), Species: gameapi.ArchaicHominin},
			gameapi.Band{ID: gameapi.BandID(id), Species: gameapi.HomoSapiens},
		)
	}
	return bands
}
