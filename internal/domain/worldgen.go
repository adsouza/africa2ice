package domain

import "math"

const (
	HighlandElevationKm = 1.0
	RiverCorridorBonus  = 0.35
	RiverAdjacentBonus  = 0.20
	ContinentalityMax   = 0.20
	ContinentalityRange = 8
	OrographicBonus     = 0.10
)

type TileGeography struct {
	ID TileID
	X  int
	Y  int
	GeoPoint
	Land           bool
	Coastal        bool
	Region         Region
	ElevationKm    float64
	NaturalShelter float64
	RiverCorridor  bool
	BaseMoisture   float64
}

type WorldGenerator struct{}

func (WorldGenerator) Generate() (*Grid, error) {
	grid := &Grid{}
	for y := range MapHeight {
		for x := range MapWidth {
			id, _ := TileIDAt(x, y)
			point, _ := GeoAt(x, y)
			center := scaledTileCenter(x, y)
			land := false
			for _, feature := range landPolygons {
				if polygonContains(feature.points, center) {
					land = true
					break
				}
			}
			if land {
				for _, feature := range waterPolygons {
					if polygonContains(feature.points, center) {
						land = false
						break
					}
				}
			}
			tile := TileGeography{ID: id, X: x, Y: y, GeoPoint: point, Land: land}
			if land {
				region, ok := resolveRegion(x, y, point)
				if !ok {
					return nil, ErrInvalidCoordinate
				}
				tile.Region = region
				for _, feature := range highlands {
					if polygonContains(feature.points, center) && feature.heightKm > tile.ElevationKm {
						tile.ElevationKm = feature.heightKm
					}
				}
				for _, feature := range shelters {
					if ellipseContains(feature, x, y) && feature.rating > tile.NaturalShelter {
						tile.NaturalShelter = feature.rating
					}
				}
			}
			grid.tiles[id] = tile
		}
	}

	riverMask := rasterizeRivers()
	for id := range TileCount {
		if grid.tiles[id].Land && riverMask[id] {
			grid.tiles[id].RiverCorridor = true
		}
	}
	if err := grid.installEscarpments(); err != nil {
		return nil, err
	}
	grid.deriveCoasts()
	grid.deriveMoisture(riverMask)
	if err := grid.deriveBiomeHistory(); err != nil {
		return nil, err
	}
	return grid, nil
}

func ellipseContains(feature shelterFeature, x, y int) bool {
	dx, dy := int64(x-feature.cx), int64(y-feature.cy)
	rx, ry := int64(feature.rx), int64(feature.ry)
	return dx*dx*ry*ry+dy*dy*rx*rx <= rx*rx*ry*ry
}

func rasterizeRivers() [TileCount]bool {
	var result [TileCount]bool
	for _, river := range rivers {
		for segment := 1; segment < len(river.points); segment++ {
			a, _ := ProjectGeo(river.points[segment-1])
			b, _ := ProjectGeo(river.points[segment])
			for id := range TileCount {
				x, y, _ := TileXY(TileID(id))
				if pointSegmentDistanceSquared(float64(x), float64(y), a.X, a.Y, b.X, b.Y) <= 0.36 {
					result[id] = true
				}
			}
		}
	}
	return result
}

func pointSegmentDistanceSquared(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	lengthSquared := float64(dx*dx) + float64(dy*dy)
	if lengthSquared == 0 {
		x, y := px-ax, py-ay
		return float64(x*x) + float64(y*y)
	}
	t := (float64((px-ax)*dx) + float64((py-ay)*dy)) / lengthSquared
	t = math.Max(0, math.Min(1, t))
	x, y := px-(ax+float64(t*dx)), py-(ay+float64(t*dy))
	return float64(x*x) + float64(y*y)
}

type moistureAnchor struct{ latitude, moisture float64 }

var moistureAnchors = [...]moistureAnchor{{0, 0.95}, {8, 0.80}, {15, 0.45}, {25, 0.12}, {35, 0.35}, {50, 0.60}, {65, 0.35}, {72, 0.25}}

func zonalMoisture(latitude float64) float64 {
	latitude = math.Abs(latitude)
	for i := 1; i < len(moistureAnchors); i++ {
		left, right := moistureAnchors[i-1], moistureAnchors[i]
		if latitude <= right.latitude {
			fraction := (latitude - left.latitude) / (right.latitude - left.latitude)
			return left.moisture + float64(fraction*(right.moisture-left.moisture))
		}
	}
	return moistureAnchors[len(moistureAnchors)-1].moisture
}

func (g *Grid) deriveCoasts() {
	for id := range TileCount {
		tile := &g.tiles[id]
		if !tile.Land {
			continue
		}
		for dy := -1; dy <= 1 && !tile.Coastal; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := tile.X+dx, tile.Y+dy
				neighbor, err := TileIDAt(nx, ny)
				if err != nil || !g.tiles[neighbor].Land {
					tile.Coastal = true
					break
				}
			}
		}
	}
}

func (g *Grid) deriveMoisture(rivers [TileCount]bool) {
	for id := range TileCount {
		tile := &g.tiles[id]
		if !tile.Land {
			continue
		}
		bonus := 0.0
		if rivers[id] {
			bonus = RiverCorridorBonus
		} else if g.hasAdjacentRiver(tile.X, tile.Y, rivers) {
			bonus = RiverAdjacentBonus
		}
		distance := g.waterDistance(tile.X, tile.Y, rivers)
		fraction := float64(distance) / ContinentalityRange
		if fraction > 1 {
			fraction = 1
		}
		penalty := float64(ContinentalityMax * fraction)
		orographic := 0.0
		if tile.ElevationKm > HighlandElevationKm {
			orographic = OrographicBonus
		}
		tile.BaseMoisture = clamp01(zonalMoisture(tile.Latitude) + bonus - penalty + orographic)
	}
}

func (g *Grid) hasAdjacentRiver(x, y int, rivers [TileCount]bool) bool {
	for _, delta := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		id, err := TileIDAt(x+delta[0], y+delta[1])
		if err == nil && rivers[id] {
			return true
		}
	}
	return false
}

func (g *Grid) waterDistance(x, y int, rivers [TileCount]bool) int {
	for distance := 0; distance <= ContinentalityRange; distance++ {
		for dy := -distance; dy <= distance; dy++ {
			for dx := -distance; dx <= distance; dx++ {
				if absInt(dx) != distance && absInt(dy) != distance {
					continue
				}
				id, err := TileIDAt(x+dx, y+dy)
				if err != nil || !g.tiles[id].Land || rivers[id] {
					return distance
				}
			}
		}
	}
	return ContinentalityRange
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
