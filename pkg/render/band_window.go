package render

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const maxVisibleSapiensBandRows = 5

// sapiensBandWindow is a bounded, allocation-free page into Frame.Bands. The
// frame may interleave computer-controlled archaic bands, so the stored values
// are source-slice indices rather than a contiguous range.
type sapiensBandWindow struct {
	indices [maxVisibleSapiensBandRows]int
	count   int
	first   int
	total   int
}

func (window sapiensBandWindow) label() string {
	if window.total <= maxVisibleSapiensBandRows {
		return fmt.Sprintf("Sapiens bands: %d", window.total)
	}
	return fmt.Sprintf("Sapiens bands %d–%d/%d", window.first+1, window.first+window.count, window.total)
}

func visibleSapiensBandWindow(bands []gameapi.Band, selectedBand gameapi.BandID) sapiensBandWindow {
	selectedOrdinal := -1
	window := sapiensBandWindow{}
	for _, band := range bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		if band.ID == selectedBand {
			selectedOrdinal = window.total
		}
		window.total++
	}
	if selectedOrdinal >= 0 {
		window.first = selectedOrdinal / maxVisibleSapiensBandRows * maxVisibleSapiensBandRows
	}

	ordinal := 0
	for index, band := range bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		if ordinal >= window.first && ordinal < window.first+maxVisibleSapiensBandRows {
			window.indices[window.count] = index
			window.count++
		}
		ordinal++
		if window.count == maxVisibleSapiensBandRows {
			break
		}
	}
	return window
}
