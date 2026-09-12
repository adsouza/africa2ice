package ui

import (
	"reflect"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

// NearbyLakeHistoryNotes only runs across completed forward turns. Loading a
// save or revealing a lake does not replay an old historical change. Proximity
// means the same or an adjacent grid cell, about one regional tile away.
func NearbyLakeHistoryNotes(before, after *gameapi.Frame) []render.FieldNote {
	if before == nil || after == nil || after.Turn <= before.Turn || after.YearBP >= before.YearBP {
		return nil
	}
	var notes []render.FieldNote
	for _, entry := range lakeHistory {
		yearBP := gameapi.LakeStageStart(entry.stage)
		if before.YearBP <= yearBP || after.YearBP > yearBP || gameapi.LakeStageAt(entry.name, after.YearBP) != entry.stage || !visibleLakeChange(before, after, entry.name) {
			continue
		}
		// Contraction can remove the nearby fragment entirely. Use both
		// shores, but only bands still alive in the accepted frame.
		if !sapiensNearLake(after, entry.name, after.Tiles) && !sapiensNearLake(after, entry.name, before.Tiles) {
			continue
		}
		effect := "The map now shows a different schematic shoreline for this lake. The date and outline summarize a broad historical interval; surrounding tiles keep their movement, resource, and habitability rules."
		notes = append(notes, render.FieldNote{
			Topic:        entry.change + " · " + entry.name,
			Introduction: "One of your sapiens bands was nearby when the mapped shoreline changed.",
			Context:      entry.context, GameEffect: effect,
			Hint:       "Inspect nearby tiles for their current water, food, and capacity before choosing a destination. The shoreline overlay does not forecast those values.",
			References: entry.references,
		})
	}
	return notes
}

func sapiensNearLake(frame *gameapi.Frame, name string, shores []gameapi.Tile) bool {
	for _, band := range frame.Bands {
		if band.Species != gameapi.HomoSapiens || band.Population == 0 || int(band.TileID) >= len(frame.Tiles) {
			continue
		}
		camp := frame.Tiles[band.TileID]
		for _, tile := range shores {
			if !tile.Explored || !strings.Contains(tile.NearbyLake, name) {
				continue
			}
			if tile.X >= camp.X-1 && tile.X <= camp.X+1 && tile.Y >= camp.Y-1 && tile.Y <= camp.Y+1 {
				return true
			}
		}
	}
	return false
}

// Compare only geography already visible on both frames. Newly explored lake
// fragments alone are not evidence of expansion, even at a stage boundary.
func visibleLakeChange(before, after *gameapi.Frame, name string) bool {
	for id, tile := range after.Tiles {
		if id >= len(before.Tiles) || !tile.Explored || !before.Tiles[id].Explored {
			continue
		}
		old := lakePoints(before.Tiles[id], name)
		current := lakePoints(tile, name)
		if !reflect.DeepEqual(old, current) {
			return true
		}
	}
	return false
}

func lakePoints(tile gameapi.Tile, name string) []gameapi.LakePoint {
	for _, lake := range tile.Lakes {
		if lake.Name == name {
			return lake.Points
		}
	}
	return nil
}
