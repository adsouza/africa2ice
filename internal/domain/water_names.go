package domain

// WaterBodyName uses the same authored polygons as terrain generation. These
// presentation labels do not classify land or change any simulation rules.
func (grid *Grid) WaterBodyName(id TileID) string {
	tile, ok := grid.Tile(id)
	if !ok || tile.Land {
		return ""
	}
	center := scaledTileCenter(tile.X, tile.Y)
	for _, feature := range waterPolygons {
		if !polygonContains(feature.points, center) {
			continue
		}
		switch feature.id {
		case "Mediterranean":
			return "Mediterranean Sea"
		case "RedSea":
			return "Red Sea"
		case "GulfOfAden":
			return "Gulf of Aden"
		case "PersianGulf":
			return "Persian Gulf"
		case "Caspian":
			return "Caspian Sea"
		case "BlackSea":
			return "Black Sea"
		case "NorthWallaceaGap", "SouthWallaceaGap":
			return "Wallacea"
		case "BeringStrait":
			return "Bering Strait"
		}
	}
	return oceanName(tile)
}

// The crop stops at 48°S, north of the Southern Ocean. The Indian/Pacific
// boundary follows a coarse Sunda/Australia outline and the Tasmania meridian;
// these display boundaries never participate in the terrain rasterization.
var indianOceanLabelBoundary = []coordinate{
	c(20*10, -48*10), c(147*10, -48*10), c(147*10, -44*10),
	c(130*10, -12*10), c(115*10, -9*10), c(105*10, -6*10),
	c(103*10, 2*10), c(100*10, 30*10), c(20*10, 30*10),
}

func oceanName(tile TileGeography) string {
	// Retain the Norwegian Sea's Atlantic connection west of the Barents
	// Sea, while the northern Eurasian/Alaskan coast belongs to the Arctic.
	if tile.Latitude >= 70 || (tile.Latitude >= 66 && tile.Longitude >= 40) {
		return "Arctic Ocean"
	}
	if tile.Longitude < 20 || (tile.Latitude >= 30 && tile.Longitude < 40) {
		return "Atlantic Ocean"
	}
	if polygonContains(indianOceanLabelBoundary, scaledTileCenter(tile.X, tile.Y)) {
		return "Indian Ocean"
	}
	return "Pacific Ocean"
}
