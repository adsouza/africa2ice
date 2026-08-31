package domain

type StartingAnchor struct {
	Species    Species
	Region     Region
	Name       string
	Point      GeoPoint
	Population Population
}

var StartingAnchors = [...]StartingAnchor{
	{Species: HomoSapiens, Region: EastAfrica, Name: "Afar", Point: g(41.0, 11.5), Population: 120},
	{Species: HomoSapiens, Region: EastAfrica, Name: "Lake Turkana", Point: g(36.0, 3.5), Population: 120},
	{Species: HomoSapiens, Region: EastAfrica, Name: "Lake Victoria Rift", Point: g(33.0, -1.0), Population: 120},
	{Species: HomoSapiens, Region: EastAfrica, Name: "Southern East African Rift", Point: g(35.0, -7.0), Population: 120},
	{Species: ArchaicHominin, Region: Levant, Name: "Northern Levant", Point: g(36.0, 34.5), Population: 120},
	{Species: ArchaicHominin, Region: Frangistan, Name: "Balkans", Point: g(22.0, 43.0), Population: 60},
	{Species: ArchaicHominin, Region: Frangistan, Name: "Iberia", Point: g(-4.0, 40.0), Population: 60},
	{Species: ArchaicHominin, Region: Siberia, Name: "Denisova Cave (Altai)", Point: g(84.68, 51.40), Population: 12},
	{Species: ArchaicHominin, Region: SoutheastAsia, Name: "Tam Pa Ling", Point: g(103.40, 20.20), Population: 12},
	{Species: ArchaicHominin, Region: EastAsia, Name: "Harbin (Denisovan)", Point: g(126.63, 45.75), Population: 90},
}

var StartingTileIDs = [len(StartingAnchors)]TileID{3098, 3480, 3671, 3960, 1944, 1459, 1639, 909, 2645, 1407}

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
