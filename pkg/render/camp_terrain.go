package render

import (
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type campTerrainColors struct {
	ground, distant, ridge, plants, stones color.NRGBA
}

func campTerrainPalette(tile gameapi.Tile) campTerrainColors {
	palette := campTerrainColors{
		ground: color.NRGBA{R: 39, G: 36, B: 28, A: 255}, distant: color.NRGBA{R: 32, G: 40, B: 41, A: 255},
		ridge: color.NRGBA{R: 25, G: 32, B: 30, A: 255}, plants: color.NRGBA{R: 16, G: 27, B: 25, A: 255}, stones: color.NRGBA{R: 61, G: 56, B: 44, A: 255},
	}
	switch tile.Biome {
	case gameapi.RiverineWoodland:
		palette.ground = color.NRGBA{R: 27, G: 35, B: 29, A: 255}
		palette.plants = color.NRGBA{R: 11, G: 27, B: 25, A: 255}
	case gameapi.CoastalShrubland:
		palette.ground = color.NRGBA{R: 47, G: 45, B: 35, A: 255}
		palette.ridge = color.NRGBA{R: 33, G: 44, B: 47, A: 255}
	case gameapi.MountainousHighlands:
		palette.ground = color.NRGBA{R: 39, G: 42, B: 43, A: 255}
		palette.distant = color.NRGBA{R: 43, G: 53, B: 64, A: 255}
		palette.ridge = color.NRGBA{R: 29, G: 37, B: 45, A: 255}
		palette.stones = color.NRGBA{R: 70, G: 74, B: 74, A: 255}
	case gameapi.SemiAridDesert:
		palette.ground = color.NRGBA{R: 62, G: 46, B: 31, A: 255}
		palette.distant = color.NRGBA{R: 59, G: 48, B: 40, A: 255}
		palette.ridge = color.NRGBA{R: 46, G: 36, B: 29, A: 255}
		palette.plants = color.NRGBA{R: 35, G: 34, B: 27, A: 255}
	case gameapi.GlacialTundra:
		palette.ground = color.NRGBA{R: 43, G: 48, B: 45, A: 255}
		palette.distant = color.NRGBA{R: 45, G: 57, B: 67, A: 255}
		palette.ridge = color.NRGBA{R: 32, G: 43, B: 49, A: 255}
		palette.plants = color.NRGBA{R: 30, G: 38, B: 37, A: 255}
		palette.stones = color.NRGBA{R: 71, G: 77, B: 78, A: 255}
	}
	if tile.LocalTemperatureC <= 0 {
		palette.ground = color.NRGBA{R: 67, G: 77, B: 81, A: 255}
	}
	return palette
}

func drawCampLandscape(c logicalCanvas, tile gameapi.Tile, band gameapi.BandID) {
	for y := 0; y < 720; y += 6 {
		f := float64(y) / 720
		vector.FillRect(c, 0, float32(y), 1280, 6, color.NRGBA{R: uint8(8 + 15*f), G: uint8(16 + 10*f), B: uint8(27 + 8*f), A: 255}, false)
	}
	// A band-stable sky and tile-driven scenery use no campaign randomness.
	for i := 0; i < 75; i++ {
		x := 45 + 1190*unitNoise(hashTile(i, int(band), 1))
		y := 130 + 165*unitNoise(hashTile(i, int(band), 2))
		vector.FillCircle(c, float32(x), float32(y), float32(.6+unitNoise(hashTile(i, 0, 3))), color.NRGBA{R: 166, G: 185, B: 194, A: 150}, false)
	}
	campEllipse(c, 1040, 190, 24, 24, color.NRGBA{R: 176, G: 187, B: 181, A: 255})
	campEllipse(c, 1031, 183, 24, 24, color.NRGBA{R: 13, G: 19, B: 29, A: 255})
	palette := campTerrainPalette(tile)
	drawCampHorizon(c, tile, palette)
	campPolygon(c, palette.ground, 0, 450, 200, 424, 425, 446, 640, 424, 880, 450, 1120, 425, 1280, 451, 1280, 720, 0, 720)
	drawCampWater(c, tile)
	drawCampPlants(c, tile, palette.plants)
	// A small windbreak and natural rock shelter keep the camp on dry ground.
	campPolygon(c, color.NRGBA{R: 46, G: 43, B: 39, A: 255}, 130, 432, 222, 330, 310, 424)
	campPolygon(c, color.NRGBA{R: 12, G: 19, B: 22, A: 255}, 182, 428, 225, 353, 268, 426)
	vector.StrokeLine(c, 225, 331, 320, 444, 4, color.NRGBA{R: 77, G: 65, B: 50, A: 255}, false)
	if tile.NaturalShelter > 0 {
		campPolygon(c, palette.ridge, 1030, 443, 1020, 381, 1090, 344, 1170, 359, 1230, 443)
	}
	drawCampGroundDetail(c, tile, palette)
	for y := 580; y < 720; y += 4 {
		vector.FillRect(c, 0, float32(y), 1280, 4, color.NRGBA{R: 5, G: 9, B: 14, A: uint8(min(255, (y-580)*3))}, false)
	}
}

func drawCampHorizon(c logicalCanvas, tile gameapi.Tile, palette campTerrainColors) {
	if tile.Biome == gameapi.CoastalShrubland {
		vector.FillRect(c, 0, 326, 1280, 130, color.NRGBA{R: 23, G: 44, B: 57, A: 255}, false)
		for i := 0; i < 12; i++ {
			y := float32(334 + i*8)
			x := float32(1000 - i*6)
			vector.StrokeLine(c, x, y, x+float32(50+i*13), y, 1.5, color.NRGBA{R: 80, G: 101, B: 108, A: 90}, false)
		}
		for i := 0; i < 9; i++ {
			x, y := float32(350+i%3*195), float32(344+i*7)
			vector.StrokeLine(c, x, y, x+float32(52+i%3*20), y, 1, color.NRGBA{R: 68, G: 88, B: 96, A: 110}, false)
		}
		campPolygon(c, palette.ridge, 0, 299, 140, 316, 320, 363, 0, 392)
		campPolygon(c, palette.distant, 1280, 330, 1140, 341, 1030, 372, 1280, 387)
		return
	}
	if tile.Biome == gameapi.SemiAridDesert {
		campEllipse(c, 200, 430, 490, 98, palette.distant)
		campEllipse(c, 1050, 455, 680, 136, palette.distant)
		campEllipse(c, 550, 469, 620, 95, palette.ridge)
		campEllipse(c, 1200, 475, 370, 71, palette.ridge)
		return
	}
	profile := []float64{361, 343, 356, 337, 367, 353, 361, 342, 360, 340, 365}
	switch tile.Biome {
	case gameapi.MountainousHighlands:
		profile = []float64{340, 251, 309, 216, 332, 280, 318, 239, 330, 265, 344}
	case gameapi.GlacialTundra:
		profile = []float64{358, 318, 334, 298, 354, 335, 352, 303, 344, 325, 362}
	}
	height := .85 + .15*math.Min(4, math.Max(0, tile.ElevationKm))
	points := make([]float64, 0, len(profile)*2+4)
	for i, y := range profile {
		points = append(points, float64(i)*128, 390-(390-y)*height)
	}
	points = append(points, 1280, 470, 0, 470)
	campPolygon(c, palette.distant, points...)
	if tile.LocalTemperatureC <= 0 && (tile.Biome == gameapi.MountainousHighlands || tile.Biome == gameapi.GlacialTundra) {
		for _, i := range []int{1, 3, 7, 9} {
			x, y := points[i*2], points[i*2+1]
			campPolygon(c, color.NRGBA{R: 104, G: 121, B: 128, A: 255}, x, y, x+26, y+27, x+5, y+20, x-25, y+28)
		}
	}
	campPolygon(c, palette.ridge, 0, 399, 190, 370, 360, 400, 540, 367, 780, 402, 1050, 375, 1280, 395, 1280, 500, 0, 500)
}

func drawCampWater(c logicalCanvas, tile gameapi.Tile) {
	water := color.NRGBA{R: 27, G: 50, B: 61, A: 255}
	if tile.Biome == gameapi.RiverineWoodland {
		campPolygon(c, water, 910, 372, 944, 372, 1070, 405, 1090, 434, 1070, 462, 1200, 502, 1280, 510, 1280, 565, 1150, 531, 1024, 470, 1050, 430, 1030, 414)
		vector.StrokeLine(c, 1072, 448, 1095, 478, 2, color.NRGBA{R: 68, G: 85, B: 89, A: 170}, false)
	}
	if tile.Biome != gameapi.CoastalShrubland && (tile.NearbyLake != "" || len(tile.Lakes) > 0) {
		campEllipse(c, 145, 427, 205, 24, water)
		for i := 0; i < 4; i++ {
			vector.StrokeLine(c, float32(60+i*20), float32(414+i*7), float32(155+i*32), float32(414+i*7), 1, color.NRGBA{R: 81, G: 100, B: 108, A: 150}, false)
		}
	}
}

func drawCampPlants(c logicalCanvas, tile gameapi.Tile, plants color.NRGBA) {
	vegetation := clampRender(tile.VegetationIndex)
	if tile.Biome == gameapi.RiverineWoodland {
		count := 9 + int(vegetation*7)
		for i := 0; i < count; i++ {
			x := float64(i)*1280/float64(count-1) + 10*unitNoise(hashTile(i, int(tile.ID), 21))
			y := 430 + 12*unitNoise(hashTile(i, int(tile.ID), 22))
			height := 112 + 68*unitNoise(hashTile(i, int(tile.ID), 23))
			vector.StrokeLine(c, float32(x), float32(y), float32(x-4), float32(y-height), 9, plants, false)
			campEllipse(c, x, y-height, 55, 37, plants)
			campEllipse(c, x-30, y-height+22, 48, 29, plants)
			campEllipse(c, x+31, y-height+20, 46, 31, plants)
		}
		return
	}
	if tile.Biome == gameapi.Savanna {
		for i, x := range []float64{70, 1165, 300 + vegetation*90} {
			y, height := 442-float64(i)*13, 103+float64(i%2)*28
			vector.StrokeLine(c, float32(x), float32(y), float32(x+12), float32(y-height), 7, plants, false)
			vector.StrokeLine(c, float32(x+8), float32(y-height+22), float32(x-28), float32(y-height-3), 4, plants, false)
			campEllipse(c, x+10, y-height-5, 65, 14, plants)
		}
	}
	count := 10 + int(vegetation*12)
	if tile.Biome == gameapi.SemiAridDesert {
		count = 4 + int(vegetation*4)
	}
	for i := 0; i < count; i++ {
		x := 20 + 1240*unitNoise(hashTile(i, int(tile.ID), 24))
		y := 452 + 24*unitNoise(hashTile(i, int(tile.ID), 25))
		if tile.Biome == gameapi.CoastalShrubland {
			campEllipse(c, x, y-10, 23, 13, plants)
			campEllipse(c, x+16, y-6, 17, 10, plants)
		} else {
			for j := 0; j < 4; j++ {
				vector.StrokeLine(c, float32(x), float32(y), float32(x+float64(j*7-12)), float32(y-float64(10+j%2*12)), 2, plants, false)
			}
		}
	}
}

func drawCampGroundDetail(c logicalCanvas, tile gameapi.Tile, palette campTerrainColors) {
	rocky := tile.Biome == gameapi.MountainousHighlands || tile.Biome == gameapi.GlacialTundra
	for i := 0; i < 40; i++ {
		x := 30 + 1220*unitNoise(hashTile(i, int(tile.ID), 7))
		y := 485 + 115*unitNoise(hashTile(i, int(tile.ID), 8))
		size := 2 + 4*unitNoise(hashTile(i, int(tile.ID), 9))
		if rocky {
			campPolygon(c, palette.stones, x-size*2, y, x-size, y-size*2, x+size, y-size*2.6, x+size*2, y)
		} else {
			campEllipse(c, x, y, size, 1.3, palette.stones)
		}
	}
	if tile.LocalTemperatureC <= 0 {
		for i := 0; i < 7; i++ {
			x := 60 + 1150*unitNoise(hashTile(i, int(tile.ID), 26))
			y := 494 + 84*unitNoise(hashTile(i, int(tile.ID), 27))
			// Keep the hearth clear even when the surrounding ground is snowy.
			if x > 550 && x < 730 {
				continue
			}
			campEllipse(c, x, y, 40+float64(i%3)*25, 4+float64(i%3)*3, color.NRGBA{R: 106, G: 120, B: 124, A: 160})
		}
	}
}
