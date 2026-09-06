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
	glyph   glyphPainter
}

// mapLegendEntries is a method, not a package function, so it can reach
// scene's painter table: the legend teaches the same pictograph a biome's
// tiles carry, rather than a second, disconnected vocabulary.
func (scene *MapScene) mapLegendEntries(aridity float64) [9]mapLegendEntry {
	return [9]mapLegendEntry{
		{label: "Riverine woodland", meaning: "plant food; disease", color: climateBiomeColor(gameapi.RiverineWoodland, aridity), glyph: scene.biomeGlyphs[gameapi.RiverineWoodland]},
		{label: "Savanna", meaning: "mixed food; drought", color: climateBiomeColor(gameapi.Savanna, aridity), glyph: scene.biomeGlyphs[gameapi.Savanna]},
		{label: "Coastal shrubland", meaning: "aquatic food; storms", color: climateBiomeColor(gameapi.CoastalShrubland, aridity), glyph: scene.biomeGlyphs[gameapi.CoastalShrubland]},
		{label: "Mountain highlands", meaning: "low capacity; cold/falls", color: climateBiomeColor(gameapi.MountainousHighlands, aridity), glyph: scene.biomeGlyphs[gameapi.MountainousHighlands]},
		{label: "Semi-arid desert", meaning: "little food/water; heat", color: climateBiomeColor(gameapi.SemiAridDesert, aridity), glyph: scene.biomeGlyphs[gameapi.SemiAridDesert]},
		{label: "Glacial tundra", meaning: "little plant food; freezing", color: climateBiomeColor(gameapi.GlacialTundra, aridity), glyph: scene.biomeGlyphs[gameapi.GlacialTundra]},
		{label: "Water", meaning: "cannot be occupied", color: EpochGrade(aridity).Water, glyph: scene.waterGlyph},
		{label: "Unknown", meaning: "not yet explored", color: unexploredTileColor},
		{label: "Escarpment", meaning: "impassable edge", color: escarpmentColor, edge: true},
	}
}
