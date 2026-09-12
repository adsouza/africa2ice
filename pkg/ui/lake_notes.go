package ui

import (
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
		if before.YearBP <= entry.yearBP || after.YearBP > entry.yearBP || !sapiensNearLake(after, entry.name) {
			continue
		}
		effect := "Historical context: lake-size changes are not simulated here. The outline stays schematic; this note does not change movement, resources, or habitability."
		if entry.yearBP == 70000 {
			effect = "The schematic Lake Lisan outline now appears on explored tiles. Surrounding land keeps its existing movement, resource, and habitability rules."
		}
		notes = append(notes, render.FieldNote{
			Topic:        entry.change + " · " + entry.name,
			Introduction: "One of your sapiens bands was nearby as the campaign entered this lake-history interval.",
			Context:      entry.context, GameEffect: effect,
			Hint:       "Inspect nearby tiles for their current water, food, and capacity before choosing a destination. The historical shoreline shift is not a forecast of those values.",
			References: entry.references,
		})
	}
	return notes
}

func sapiensNearLake(frame *gameapi.Frame, name string) bool {
	for _, band := range frame.Bands {
		if band.Species != gameapi.HomoSapiens || band.Population == 0 || int(band.TileID) >= len(frame.Tiles) {
			continue
		}
		camp := frame.Tiles[band.TileID]
		for _, tile := range frame.Tiles {
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
