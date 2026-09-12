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
	return "Open ocean"
}
