package render

import (
	"image/color"
	"strconv"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// Interbreeding is the one spatial action whose availability depends on another
// band being in the same place, so it is the one the player cannot infer from
// their own band alone. These helpers turn the projected state into the three
// distinct things the HUD needs to say about it.

type interbreedState uint8

const (
	interbreedUnavailable interbreedState = iota
	interbreedAvailable
	interbreedAccepted
)

type interbreedSummary struct {
	state    interbreedState
	targetID gameapi.BandID
}

// interbreedStatus reports what the HUD should say about a band's interbreeding
// option. Only a sapiens band is ever the actor, and an accepted intent is
// reported ahead of the spent spatial action it necessarily implies.
func interbreedStatus(band gameapi.Band) interbreedSummary {
	if band.Species != gameapi.HomoSapiens {
		return interbreedSummary{}
	}
	if band.HasInterbreedTarget {
		return interbreedSummary{state: interbreedAccepted, targetID: band.InterbreedTargetID}
	}
	if band.SpatialActionUsed || len(band.InterbreedCandidateIDs) == 0 {
		return interbreedSummary{}
	}
	return interbreedSummary{state: interbreedAvailable, targetID: band.InterbreedCandidateIDs[0]}
}

// spatialControlHint lists only the keys that will actually do something. An
// advertised control that silently does nothing is worse than an absent one.
func spatialControlHint(status interbreedSummary) string {
	if status.state == interbreedAvailable {
		return "N: split  ·  I: interbreed"
	}
	return "N: split"
}

// interbreedPanelLine describes the band's interbreeding state for the band
// panel, or returns "" when there is nothing worth a line.
func interbreedPanelLine(status interbreedSummary) string {
	switch status.state {
	case interbreedAvailable:
		return "Archaic band here — press I to interbreed"
	case interbreedAccepted:
		return "Interbreeding with archaic band " + strconv.FormatUint(uint64(status.targetID), 10)
	default:
		return ""
	}
}

// interbreedCandidateTiles returns the tiles holding an archaic band that the
// given band could interbreed with, so the map can mark them.
func interbreedCandidateTiles(frame *gameapi.Frame, band gameapi.Band) map[gameapi.TileID]bool {
	status := interbreedStatus(band)
	if frame == nil || status.state == interbreedUnavailable {
		return nil
	}
	wanted := make(map[gameapi.BandID]bool, len(band.InterbreedCandidateIDs)+1)
	if status.state == interbreedAccepted {
		wanted[status.targetID] = true
	} else {
		for _, id := range band.InterbreedCandidateIDs {
			wanted[id] = true
		}
	}
	tiles := make(map[gameapi.TileID]bool, len(wanted))
	for _, other := range frame.Bands {
		if wanted[other.ID] {
			tiles[other.TileID] = true
		}
	}
	return tiles
}

// interbreedMarkerColor distinguishes an interbreeding-related mark from the
// gold sapiens and rust archaic band markers.
var interbreedMarkerColor = color.RGBA{R: 186, G: 148, B: 232, A: 255}
