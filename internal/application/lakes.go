package application

import (
	"math"
	"strings"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// These schematic footprints are presentation geography, not water-tile masks.
// At 96x64 a narrow rift lake is much smaller than a cell. Treating every cell
// it touches as water would erase shores and starting camps. See docs/LAKES.md
// for scope, chronology and research; these are not dated shoreline surveys.
type lakeFeature struct {
	name   string
	fromBP int
	stage  gameapi.LakeStage
	ring   []gameapi.LakePoint // Longitude/latitude until projected below.
}

var lakeFeatures = []lakeFeature{
	{"Lake Baikal", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 103.7, Y: 51.7}, {X: 104.8, Y: 51.5}, {X: 106.2, Y: 52.2}, {X: 107.3, Y: 53.3}, {X: 109.8, Y: 55.7}, {X: 109.4, Y: 55.9}, {X: 108.1, Y: 54.5}, {X: 106.7, Y: 53.5}, {X: 105.4, Y: 52.6}}},
	{"Lake Tanganyika", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 29.1, Y: -3.3}, {X: 29.6, Y: -3.4}, {X: 29.8, Y: -4.8}, {X: 30.4, Y: -6.1}, {X: 31.2, Y: -8.5}, {X: 30.6, Y: -8.8}, {X: 30.0, Y: -7.2}, {X: 29.3, Y: -5.9}, {X: 29.1, Y: -4.5}}},
	{"Lake Victoria", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 31.7, Y: -0.2}, {X: 32.4, Y: 0.4}, {X: 33.4, Y: 0.3}, {X: 34.1, Y: -0.4}, {X: 34.0, Y: -1.2}, {X: 33.4, Y: -1.9}, {X: 32.8, Y: -2.5}, {X: 32.0, Y: -2.3}, {X: 31.6, Y: -1.4}}},
	{"Lake Turkana", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 35.9, Y: 4.7}, {X: 36.2, Y: 4.5}, {X: 36.5, Y: 3.6}, {X: 36.7, Y: 2.5}, {X: 36.6, Y: 2.4}, {X: 36.2, Y: 2.8}, {X: 35.9, Y: 3.8}}},
	{"Issyk-Kul", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 76.1, Y: 42.5}, {X: 76.8, Y: 42.7}, {X: 77.8, Y: 42.7}, {X: 78.3, Y: 42.6}, {X: 78.4, Y: 42.4}, {X: 77.5, Y: 42.2}, {X: 76.6, Y: 42.3}}},
	{"Lake Qinghai", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 99.6, Y: 37.1}, {X: 100.2, Y: 37.3}, {X: 100.8, Y: 37.1}, {X: 100.7, Y: 36.6}, {X: 100.2, Y: 36.5}, {X: 99.8, Y: 36.7}}},
	{"Lake Van", 80000, gameapi.LakeUnstaged, []gameapi.LakePoint{{X: 42.2, Y: 38.5}, {X: 42.5, Y: 38.8}, {X: 43.0, Y: 39.0}, {X: 43.3, Y: 38.9}, {X: 43.0, Y: 38.7}, {X: 43.4, Y: 38.5}, {X: 42.9, Y: 38.4}, {X: 42.5, Y: 38.4}}},
}

type lakePatch struct {
	tile   gameapi.TileID
	fromBP int
	stage  gameapi.LakeStage
	shape  gameapi.LakeShape
}

var lakePatches = buildLakePatches()

func buildLakePatches() []lakePatch {
	var patches []lakePatch
	for _, lake := range append(append([]lakeFeature(nil), lakeFeatures...), changingLakeFeatures...) {
		ring := make([]gameapi.LakePoint, len(lake.ring))
		minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for i, p := range lake.ring {
			// Domain coordinates identify cell centres, not cell corners.
			x := (p.X-domain.MinLongitude)*float64(domain.MapWidth-1)/(domain.MaxLongitude-domain.MinLongitude) + 0.5
			y := (domain.MaxLatitude-p.Y)*float64(domain.MapHeight-1)/(domain.MaxLatitude-domain.MinLatitude) + 0.5
			ring[i] = gameapi.LakePoint{X: x, Y: y}
			minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
		}
		for y := max(0, int(math.Floor(minY))); y <= min(domain.MapHeight-1, int(math.Floor(maxY))); y++ {
			for x := max(0, int(math.Floor(minX))); x <= min(domain.MapWidth-1, int(math.Floor(maxX))); x++ {
				points := clipLakeRing(ring, float64(x), float64(y))
				if len(points) < 3 {
					continue
				}
				for i := range points {
					points[i].X -= float64(x)
					points[i].Y -= float64(y)
				}
				patches = append(patches, lakePatch{tile: gameapi.TileID(y*domain.MapWidth + x), fromBP: lake.fromBP, stage: lake.stage, shape: gameapi.LakeShape{Name: lake.name, Stage: lake.stage, Points: points}})
			}
		}
	}
	return patches
}

// Sutherland-Hodgman clipping confines each fragment to its own tile. Revealing
// one shore must never reveal the lake outline in neighbouring unexplored cells.
func clipLakeRing(ring []gameapi.LakePoint, x, y float64) []gameapi.LakePoint {
	points := ring
	for _, edge := range []struct {
		axis        int
		bound, sign float64
	}{{0, x, 1}, {0, x + 1, -1}, {1, y, 1}, {1, y + 1, -1}} {
		if len(points) == 0 {
			break
		}
		value := func(p gameapi.LakePoint) float64 {
			if edge.axis == 0 {
				return p.X
			}
			return p.Y
		}
		inside := func(p gameapi.LakePoint) bool { return (value(p)-edge.bound)*edge.sign >= 0 }
		output := make([]gameapi.LakePoint, 0, len(points)+2)
		previous := points[len(points)-1]
		for _, current := range points {
			if inside(current) != inside(previous) {
				t := (edge.bound - value(previous)) / (value(current) - value(previous))
				output = append(output, gameapi.LakePoint{X: previous.X + t*(current.X-previous.X), Y: previous.Y + t*(current.Y-previous.Y)})
			}
			if inside(current) {
				output = append(output, current)
			}
			previous = current
		}
		points = output
	}
	return points
}

func projectLakes(frame *gameapi.Frame) {
	for _, patch := range lakePatches {
		if frame.YearBP > patch.fromBP || (patch.stage != gameapi.LakeUnstaged && gameapi.LakeStageAt(patch.shape.Name, frame.YearBP) != patch.stage) || int(patch.tile) >= len(frame.Tiles) {
			continue
		}
		tile := &frame.Tiles[patch.tile]
		if !tile.Explored || !tile.Land {
			continue
		}
		shape := patch.shape
		shape.Points = append([]gameapi.LakePoint(nil), shape.Points...)
		tile.Lakes = append(tile.Lakes, shape)
		if tile.NearbyLake == "" {
			tile.NearbyLake = shape.Name
		} else if !strings.Contains(tile.NearbyLake, shape.Name) {
			tile.NearbyLake += "; " + shape.Name
		}
	}
}
