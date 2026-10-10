package render

import (
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// The departure uses the camp's procedural shapes, with a daylight palette
// and a shallow horizon sized for the map rather than a full-screen hearth.
func departurePalette(tile gameapi.Tile) campTerrainColors {
	p := campTerrainColors{
		ground: color.NRGBA{R: 190, G: 174, B: 115, A: 255}, distant: color.NRGBA{R: 153, G: 175, B: 151, A: 255},
		ridge: color.NRGBA{R: 124, G: 146, B: 112, A: 255}, plants: color.NRGBA{R: 82, G: 113, B: 65, A: 255}, stones: color.NRGBA{R: 155, G: 139, B: 105, A: 255},
	}
	switch tile.Biome {
	case gameapi.RiverineWoodland:
		p.ground = color.NRGBA{R: 151, G: 163, B: 110, A: 255}
		p.plants = color.NRGBA{R: 62, G: 109, B: 72, A: 255}
	case gameapi.CoastalShrubland:
		p.ground = color.NRGBA{R: 211, G: 198, B: 150, A: 255}
		p.ridge = color.NRGBA{R: 128, G: 160, B: 143, A: 255}
	case gameapi.MountainousHighlands:
		p.ground = color.NRGBA{R: 167, G: 165, B: 148, A: 255}
		p.distant = color.NRGBA{R: 142, G: 163, B: 178, A: 255}
		p.ridge = color.NRGBA{R: 114, G: 134, B: 142, A: 255}
		p.stones = color.NRGBA{R: 117, G: 121, B: 118, A: 255}
	case gameapi.SemiAridDesert:
		p.ground = color.NRGBA{R: 224, G: 194, B: 136, A: 255}
		p.distant = color.NRGBA{R: 221, G: 188, B: 133, A: 255}
		p.ridge = color.NRGBA{R: 196, G: 157, B: 107, A: 255}
		p.plants = color.NRGBA{R: 130, G: 135, B: 85, A: 255}
	case gameapi.GlacialTundra:
		p.ground = color.NRGBA{R: 181, G: 190, B: 175, A: 255}
		p.distant = color.NRGBA{R: 167, G: 187, B: 199, A: 255}
		p.ridge = color.NRGBA{R: 136, G: 158, B: 157, A: 255}
		p.plants = color.NRGBA{R: 106, G: 126, B: 104, A: 255}
	}
	return p
}

func drawDepartureLandscape(c logicalCanvas, tile gameapi.Tile) {
	p := departurePalette(tile)
	// Clear blue daylight, a pale sun, and thin clouds establish daytime even
	// where woodland canopy or a cold mountain horizon occupies much of the sky.
	for y := 0; y < departureHeight; y += 4 {
		fraction := float64(y) / departureHeight
		vector.FillRect(c, 0, float32(y), departureWidth, 4, color.NRGBA{R: uint8(156 + fraction*48), G: uint8(202 + fraction*22), B: 227, A: 255}, false)
	}
	campEllipse(c, 586, 38, 15, 15, color.NRGBA{R: 255, G: 240, B: 187, A: 255})
	for _, cloud := range [][3]float64{{166, 32, 48}, {427, 24, 33}, {658, 59, 37}} {
		campEllipse(c, cloud[0], cloud[1], cloud[2], 5, color.NRGBA{R: 245, G: 247, B: 238, A: 200})
	}
	if tile.Biome == gameapi.SemiAridDesert {
		campEllipse(c, 180, 119, 320, 47, p.distant)
		campEllipse(c, 590, 131, 410, 62, p.ridge)
	} else {
		profile := []float64{86, 71, 83, 67, 88, 74, 85}
		switch tile.Biome {
		case gameapi.MountainousHighlands:
			profile = []float64{85, 40, 73, 28, 86, 49, 84}
		case gameapi.GlacialTundra:
			profile = []float64{86, 61, 75, 51, 87, 63, 83}
		}
		points := make([]float64, 0, 18)
		height := .85 + .1*math.Max(0, math.Min(4, tile.ElevationKm))
		for i, y := range profile {
			points = append(points, float64(i)*120, 100-(100-y)*height)
		}
		points = append(points, departureWidth, 130, 0, 130)
		campPolygon(c, p.distant, points...)
		if tile.LocalTemperatureC <= 0 && (tile.Biome == gameapi.MountainousHighlands || tile.Biome == gameapi.GlacialTundra) {
			for _, i := range []int{1, 3, 5} {
				x, y := points[i*2], points[i*2+1]
				campPolygon(c, color.NRGBA{R: 238, G: 243, B: 239, A: 255}, x, y, x+16, y+18, x+2, y+12, x-16, y+18)
			}
		}
		campPolygon(c, p.ridge, 0, 103, 160, 88, 330, 105, 520, 91, departureWidth, 103, departureWidth, 140, 0, 140)
	}
	water := color.NRGBA{R: 91, G: 163, B: 184, A: 255}
	if tile.Biome == gameapi.CoastalShrubland {
		vector.FillRect(c, 0, 83, departureWidth, 39, water, false)
		for i := 0; i < 7; i++ {
			vector.StrokeLine(c, float32(340+i*39), float32(88+i*4), float32(382+i*39), float32(88+i*4), 1, color.NRGBA{R: 207, G: 233, B: 226, A: 190}, false)
		}
	}
	campPolygon(c, p.ground, 0, 122, 150, 115, 330, 125, 530, 117, departureWidth, 123, departureWidth, departureHeight, 0, departureHeight)
	if tile.Biome == gameapi.RiverineWoodland || tile.NearbyLake != "" || len(tile.Lakes) > 0 {
		campEllipse(c, 636, 118, 80, 7, water)
	}
	vegetation := clampRender(tile.VegetationIndex)
	for i := 0; i < 5; i++ {
		x := 56 + float64(i)*151
		switch tile.Biome {
		case gameapi.RiverineWoodland, gameapi.Savanna:
			height := 48 + vegetation*26 + float64(i%2)*8
			vector.StrokeLine(c, float32(x), 121, float32(x+3), float32(121-height), 4, p.plants, false)
			ry := 9.0
			if tile.Biome == gameapi.RiverineWoodland {
				ry = 21
			}
			campEllipse(c, x+3, 119-height, 31+vegetation*8, ry, p.plants)
		case gameapi.CoastalShrubland:
			campEllipse(c, x, 119, 17, 8, p.plants)
		}
	}
	for i := 0; i < 26; i++ {
		x := 25 + 670*unitNoise(hashTile(i, int(tile.ID), 61))
		y := 130 + 23*unitNoise(hashTile(i, int(tile.ID), 62))
		if tile.Biome == gameapi.MountainousHighlands {
			campPolygon(c, p.stones, x-4, y, x-2, y-5, x+3, y-4, x+6, y)
		} else {
			campEllipse(c, x, y, 3, 1, p.stones)
		}
		if tile.LocalTemperatureC <= 0 && i%4 == 0 {
			campEllipse(c, x, y, 15, 2, color.NRGBA{R: 237, G: 242, B: 233, A: 220})
		}
	}
}
