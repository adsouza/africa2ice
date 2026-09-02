package render

import (
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type mapLegendEntry struct {
	label   string
	meaning string
	color   color.RGBA
	edge    bool
}

func mapLegendEntries(aridity float64) [9]mapLegendEntry {
	return [9]mapLegendEntry{
		{label: "Riverine woodland", meaning: "plant food; disease", color: climateBiomeColor(gameapi.RiverineWoodland, aridity)},
		{label: "Savanna", meaning: "mixed food; drought", color: climateBiomeColor(gameapi.Savanna, aridity)},
		{label: "Coastal shrubland", meaning: "aquatic food; storms", color: climateBiomeColor(gameapi.CoastalShrubland, aridity)},
		{label: "Mountain highlands", meaning: "low capacity; cold/falls", color: climateBiomeColor(gameapi.MountainousHighlands, aridity)},
		{label: "Semi-arid desert", meaning: "little food/water; heat", color: climateBiomeColor(gameapi.SemiAridDesert, aridity)},
		{label: "Glacial tundra", meaning: "little plant food; freezing", color: climateBiomeColor(gameapi.GlacialTundra, aridity)},
		{label: "Water", meaning: "cannot be occupied", color: EpochGrade(aridity).Water},
		{label: "Unknown", meaning: "not yet explored", color: unexploredTileColor},
		{label: "Escarpment", meaning: "impassable edge", color: escarpmentColor, edge: true},
	}
}
