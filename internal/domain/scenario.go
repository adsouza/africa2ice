package domain

type StartingAnchor struct {
	Species Species
	Region  Region
	Name    string
	Point   GeoPoint
}

var StartingAnchors = [...]StartingAnchor{
	{HomoSapiens, EastAfrica, "Afar", g(41.0, 11.5)},
	{HomoSapiens, EastAfrica, "Lake Turkana", g(36.0, 3.5)},
	{HomoSapiens, EastAfrica, "Lake Victoria Rift", g(33.0, -1.0)},
	{HomoSapiens, EastAfrica, "Southern East African Rift", g(35.0, -7.0)},
	{ArchaicHominin, Levant, "Northern Levant", g(36.0, 34.5)},
	{ArchaicHominin, Levant, "Southern Levant", g(35.0, 31.5)},
	{ArchaicHominin, Frangistan, "Balkans", g(22.0, 43.0)},
	{ArchaicHominin, Frangistan, "Iberia", g(-4.0, 40.0)},
}

var StartingTileIDs = [len(StartingAnchors)]TileID{3098, 3480, 3671, 3960, 1944, 2040, 1459, 1639}

func ResolveStartingTiles(grid *Grid, habitat *Habitat) ([len(StartingAnchors)]TileID, error) {
	var result [len(StartingAnchors)]TileID
	used := [TileCount]bool{}
	for index, anchor := range StartingAnchors {
		projected, err := ProjectGeo(anchor.Point)
		if err != nil {
			return result, err
		}
		best, bestDistance := InvalidTileID, 0.0
		for id := range TileCount {
			geography, _ := grid.Tile(TileID(id))
			if used[id] || !geography.Land || geography.Region != anchor.Region || habitat[id].BaselineK <= 0 {
				continue
			}
			dx, dy := float64(geography.X)-projected.X, float64(geography.Y)-projected.Y
			distance := float64(dx*dx) + float64(dy*dy)
			if best == InvalidTileID || distance < bestDistance || (distance == bestDistance && TileID(id) < best) {
				best, bestDistance = TileID(id), distance
			}
		}
		if best == InvalidTileID {
			return result, ErrInvalidCoordinate
		}
		result[index], used[best] = best, true
	}
	return result, nil
}
